package gsm

import (
	"fmt"
	"go-gsm/pkg/logrus"
	"strings"
	"time"
)

type CallObserver struct {
	SerialSubject *SerialSubject
}

func NewCallObserver(subject *SerialSubject) *CallObserver {
	return &CallObserver{
		SerialSubject: subject,
	}
}

func (c *CallObserver) isCallResponse(data string) bool {
	allows := []string{"RING", "NO CARRIER"}
	for _, allow := range allows {
		if strings.Contains(data, allow) {
			return true
		}
	}
	return false
}

func (c *CallObserver) Update(data string) {
	//logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Debugf("CallObserver: %s", data)

	if !c.isCallResponse(data) {
		return
	}
	if data == "RING" {
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Incoming call detected.")
		_ = c.SerialSubject.Send("ATA")
		time.Sleep(1 * time.Second)
		command := fmt.Sprintf("AT+QAUDRD=1,\"%s.wav\",13,1", c.SerialSubject.portName)
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info(command)
		_ = c.SerialSubject.Send(command)
		return
	}

	if data == "NO CARRIER" {
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info("Call ended due to NO CARRIER or ERROR.")
		_ = c.SerialSubject.Send("AT+QAUDRD=0")
		time.Sleep(1 * time.Second)
		command := fmt.Sprintf("AT+QFDWL=\"%s.wav\";\r", c.SerialSubject.portName)
		logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Info(command)
		if err := c.SerialSubject.Send(command); err != nil {
			logrus.LogrusLoggerWithContext(c.SerialSubject.ctx).Errorf("Error downloading recording: %v", err)
		}
		return
	}
}
