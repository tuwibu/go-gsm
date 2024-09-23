package gsm

import (
	"encoding/hex"
	"fmt"
	"go-gsm/pkg/logrus"
	"regexp"
	"strings"
	"unicode/utf16"
)

type SMSObserver struct {
	SerialSubject *SerialSubject
	queue         []string
	isReading     bool
}

func NewSMSObserver(subject *SerialSubject) *SMSObserver {
	return &SMSObserver{
		SerialSubject: subject,
		queue:         []string{},
		isReading:     false,
	}
}

func (s *SMSObserver) isSMSResponse(data string) bool {
	allows := []string{"+CMTI:", "+CMGR:"}
	for _, allow := range allows {
		if strings.Contains(data, allow) {
			return true
		}
	}
	return false
}

func (s *SMSObserver) isNotify(data string) bool {
	return strings.Contains(data, "+CMTI:")
}

func (s *SMSObserver) isRead(data string) bool {
	return strings.Contains(data, "+CMGR:")
}

func (s *SMSObserver) enqueueSMS(index string) {
	s.queue = append(s.queue, index)
	if !s.isReading {
		s.processNextSMS()
	}
}

func (s *SMSObserver) processNextSMS() {
	if len(s.queue) == 0 {
		return
	}
	index := s.queue[0]
	s.queue = s.queue[1:]

	if err := s.SerialSubject.Send(fmt.Sprintf("AT+CMGR=%s", index)); err != nil {
		logrus.LogrusLoggerWithContext(s.SerialSubject.ctx).Errorf("Error reading SMS: %v", err)
	} else {
		s.isReading = true
	}
}

func (s *SMSObserver) readSMS(data string) {
	var re = regexp.MustCompile(`(?m)\+CMGR: "REC UNREAD","(.*?)",,"(.*?)"`)
	match := re.FindStringSubmatch(data)
	if len(match) < 3 {
		return
	}
	content := strings.Join(strings.Split(data, "\r\n")[1:], "\r\n")
	decode, err := decodeUCS2(content)
	if err == nil {
		content = decode
	}
	sender := match[1]
	time := match[2]
	logrus.LogrusLoggerWithContext(s.SerialSubject.ctx).Infof("SMS from %s at %s: %s", sender, time, content)
	s.isReading = false
	s.processNextSMS()
}

func decodeUCS2(inputStr string) (string, error) {
	bytes, err := hex.DecodeString(inputStr)
	if err != nil {
		return inputStr, nil
	}
	if len(bytes)%2 != 0 {
		return inputStr, nil
	}

	runes := make([]uint16, len(bytes)/2)
	for i := 0; i < len(runes); i++ {
		runes[i] = uint16(bytes[2*i])<<8 | uint16(bytes[2*i+1])
	}

	decoded := string(utf16.Decode(runes))
	return decoded, nil
}

func (s *SMSObserver) Update(data string) {
	if !s.isSMSResponse(data) {
		return
	}
	if s.isNotify(data) {
		index := strings.Replace(data, "+CMTI: \"ME\",", "", 1)
		index = strings.TrimSpace(index)
		s.enqueueSMS(index)
	}
	if s.isRead(data) {
		s.readSMS(data)
	}
}
