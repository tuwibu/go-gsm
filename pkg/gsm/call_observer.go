package gsm

import "fmt"

type CallObserver struct {
	SerialSubject *SerialSubject
}

func NewCallObserver(subject *SerialSubject) *CallObserver {
	return &CallObserver{
		SerialSubject: subject,
	}
}

func (c *CallObserver) isCallResponse(data string) bool {
	return len(data) > 0 && data[0] == 'C'
}

func (c *CallObserver) Update(data string) {
	if c.isCallResponse(data) {
		fmt.Println("Handling Call response:", data)
	}
}
