package gsm

import (
	"go-gsm/pkg/logrus"
	"strconv"
	"strings"
)

type InfoObserver struct {
	SerialSubject *SerialSubject
}

func NewInfoObserver(subject *SerialSubject) *InfoObserver {
	return &InfoObserver{
		SerialSubject: subject,
	}
}

func (u *InfoObserver) isInfoResponse(data string) bool {
	allows := []string{"+CSQ:", "+CCID:"}
	for _, allow := range allows {
		if strings.Contains(data, allow) {
			return true
		}
	}
	return false
}

func (u *InfoObserver) Update(data string) {
	if !u.isInfoResponse(data) {
		return
	}
	if strings.Contains(data, "+CSQ:") {
		// Tín hiệu mạng
		signalStr := strings.TrimSpace(strings.Split(data, "+CSQ:")[1])
		signalStr = strings.Split(signalStr, ",")[0]
		signal, err := strconv.Atoi(signalStr)
		if err != nil {
			return
		}
		signal = (signal * 5) / 31
		u.SerialSubject.signal = signal
		logrus.LogrusLoggerWithContext(u.SerialSubject.ctx).Infof("Signal strength: %d", signal)
	}
	if strings.Contains(data, "+CCID:") {
		// ICCID
		u.SerialSubject.ccid = strings.TrimSpace(strings.Split(data, "+CCID:")[1])
		logrus.LogrusLoggerWithContext(u.SerialSubject.ctx).Infof("ICCID: %s", u.SerialSubject.ccid)
	}
}
