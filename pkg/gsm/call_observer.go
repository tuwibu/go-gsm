package gsm

import (
	"go-gsm/pkg/logrus"
	"log"
	"strings"
	"time"
)

type CallObserver struct {
	SerialSubject *SerialSubject
	isRecording   bool
	checkTime     time.Time
}

func NewCallObserver(subject *SerialSubject) *CallObserver {
	return &CallObserver{
		SerialSubject: subject,
		isRecording:   false,
	}
}

// Kiểm tra phản hồi liên quan đến cuộc gọi
func (c *CallObserver) isCallResponse(data string) bool {
	//return data == "RING" || strings.HasPrefix(data, "+CLCC") || data == "OK" || data == "ERROR"
	allows := []string{"RING", "+CLCC:", "OK", "ERROR", "NO CARRIER"}
	for _, allow := range allows {
		if strings.Contains(data, allow) {
			return true
		}
	}
	return false
}

// Trả lời cuộc gọi và bắt đầu ghi âm
func (c *CallObserver) answerAndRecord() error {
	// Trả lời cuộc gọi
	if err := c.SerialSubject.Send("ATA"); err != nil {
		return err
	}
	logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Answered call.")

	// Bắt đầu ghi âm cuộc gọi (lưu tạm vào RAM)
	if err := c.SerialSubject.Send(`AT+QAUDRD=1,"RAM:call_record.wav",60`); err != nil {
		return err
	}
	c.isRecording = true
	c.checkTime = time.Now()
	logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Started recording the call.")
	return nil
}

// Dừng ghi âm và kết thúc cuộc gọi
func (c *CallObserver) stopRecordingAndHangup() error {
	// Nếu đang ghi âm, dừng lại
	if c.isRecording {
		if err := c.SerialSubject.Send("AT+QAUDRD=0"); err != nil {
			logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Errorf("Error stopping recording: %v", err)
			return err
		}
		c.isRecording = false
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Stopped recording the call.")
	}

	// Kết thúc cuộc gọi
	if err := c.SerialSubject.Send("ATH"); err != nil {
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Errorf("Error hanging up the call: %v", err)
		return err
	}
	log.Println("Hung up the call.")
	return nil
}

// Kiểm tra trạng thái cuộc gọi với AT+CLCC
func (c *CallObserver) checkCallStatus() {
	for {
		// Gửi lệnh AT+CLCC để kiểm tra trạng thái cuộc gọi
		if err := c.SerialSubject.Send("AT+CLCC"); err != nil {
			log.Println("Error sending AT+CLCC:", err)
			return
		}
		// Đợi 5 giây trước khi kiểm tra lại
		time.Sleep(5 * time.Second)
	}
}

// Xử lý phản hồi của AT+CLCC
func (c *CallObserver) handleCLCCResponse(data string) {
	// Phân tích phản hồi của +CLCC
	if strings.HasPrefix(data, "+CLCC:") {
		// Nếu phản hồi không có số điện thoại (ví dụ, số rỗng), kết thúc cuộc gọi
		// Xem chuỗi `+CLCC: 1,1,0,1,0,"",128` là kết thúc
		//if strings.Contains(data, `,"",`) {
		//	log.Println("No active calls (empty phone number), stopping recording.")
		//	//if err := c.stopRecordingAndHangup(); err != nil {
		//	//	log.Fatalln("Error stopping recording and hanging up:", err)
		//	//}
		//} else {
		//	log.Println("Active call detected, continuing recording.")
		//}
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Active call detected, continuing recording.")
	}
}

// Theo dõi dữ liệu nhận được và phản hồi tương ứng
func (c *CallObserver) Update(data string) {
	//logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Debugf("CallObserver: %s", data)

	if !c.isCallResponse(data) {
		return
	}

	// Phát hiện cuộc gọi đến
	if data == "RING" {
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Incoming call detected.")
		if err := c.answerAndRecord(); err != nil {
			logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Errorf("Error answering and recording call: %v", err)
		}

		// Khởi động quá trình kiểm tra trạng thái cuộc gọi
		go c.checkCallStatus()
		return
	}

	// Xử lý phản hồi từ AT+CLCC
	if strings.HasPrefix(data, "+CLCC:") {
		c.handleCLCCResponse(data)
		return
	}

	// Nếu phát hiện lỗi hoặc NO CARRIER, dừng ghi âm
	if data == "NO CARRIER" || data == "ERROR" {
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Call ended due to NO CARRIER or ERROR.")
		if err := c.stopRecordingAndHangup(); err != nil {
			logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Errorf("Error stopping recording and hanging up: %v", err)
		}
		return
	}

}
