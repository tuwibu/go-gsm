package gsm

import (
	"go-gsm/pkg/logrus"
	"regexp"
	"strings"
)

type USSDObserver struct {
	SerialSubject *SerialSubject
}

func NewUSSDObserver(subject *SerialSubject) *USSDObserver {
	return &USSDObserver{
		SerialSubject: subject,
	}
}

func (u *USSDObserver) isUSSDResponse(data string) bool {
	return strings.Contains(data, "+CUSD:")
}

func (u *USSDObserver) Update(data string) {
	if !u.isUSSDResponse(data) {
		return
	}
	var re = regexp.MustCompile(`(?m)(?s)\+CUSD: [01],"(.*?)"`)
	match := re.FindStringSubmatch(data)
	if len(match) > 1 {
		content := match[1]
		var rePhone = regexp.MustCompile(`(?m)(\d{10,}|\+84\d{9})`)
		matchPhone := rePhone.FindStringSubmatch(content)
		if len(matchPhone) > 1 {
			u.SerialSubject.phone = matchPhone[1]
		}
		logrus.LogrusLoggerWithContext(u.SerialSubject.ctx).Infof("USSD response: %s", content)
	}
}
