package gsm

import (
	"log"
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
	return strings.Index(data, "+CUSD:") != -1
}

func (u *USSDObserver) Update(data string) {
	if u.isUSSDResponse(data) == false {
		return
	}
	var re = regexp.MustCompile(`(?m)(?s)\+CUSD: [01],"(.*?)"`)
	match := re.FindStringSubmatch(data)
	isSuccess := strings.Index(data, "+CUSD: 1") != -1
	if len(match) > 1 {
		if isSuccess {
			log.Println("USSD response:", match[1])
		} else {
			log.Fatal("USSD response:", match[1])
		}
	}
}
