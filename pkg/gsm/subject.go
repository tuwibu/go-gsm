package gsm

import (
	"fmt"
	"go.bug.st/serial"
	"strings"
	"sync"
)

type SerialObserver interface {
	Update(data string)
}

type SerialSubject struct {
	observers []SerialObserver
	mu        sync.RWMutex
	port      serial.Port
	buffer    string
}

// GetAvailablePorts returns a list of available serial ports
func GetAvailablePorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}
	return ports, nil
}

func CreatePort(portName string, baudRate int) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}
	port, errOpen := serial.Open(portName, mode)
	return port, errOpen
}

func NewSerial(port serial.Port) *SerialSubject {
	return &SerialSubject{
		observers: make([]SerialObserver, 0),
		mu:        sync.RWMutex{},
		port:      port,
		buffer:    "",
	}
}

// Detach removes an observer from the list of observers
func (s *SerialSubject) detach(observer SerialObserver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, o := range s.observers {
		if o == observer {
			s.observers = append(s.observers[:i], s.observers[i+1:]...)
			break
		}
	}
}

// Attach adds an observer to the list of observers
func (s *SerialSubject) attach(observer SerialObserver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, observer)
}

// Notify sends a message to all observers
func (s *SerialSubject) notify(data string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, observer := range s.observers {
		observer.Update(data)
	}
}

func (s *SerialSubject) Open() error {
	s.attach(NewUSSDObserver(s))
	s.attach(NewSMSObserver(s))
	go s.read()
	// Set the modem to text mode
	if err := s.Send("AT+CMGF=1"); err != nil {
		return err
	}
	// Set the modem to notify when a new SMS is received
	if err := s.Send("AT+CNMI=2,2,0,0,0"); err != nil {
		return err
	}
	return nil
}

func (s *SerialSubject) read() {
	for {
		buf := make([]byte, 128)
		n, err := s.port.Read(buf)
		if err != nil {
			return
		}
		s.buffer += string(buf[:n])
		for {
			if idx := strings.Index(s.buffer, "\r\n\r\n"); idx != -1 {
				message := s.buffer[:idx]
				s.buffer = s.buffer[idx+4:]
				s.notify(message)
			} else {
				break
			}
		}
	}
}

// Close closes the serial port
func (s *SerialSubject) Close() error {
	errClose := s.port.Close()
	if errClose != nil {
		return errClose
	}
	return nil
}

// Send sends a message to the serial port
func (s *SerialSubject) Send(data string) error {
	_, errWrite := s.port.Write([]byte(fmt.Sprintf("%s\r\n", data)))
	if errWrite != nil {
		return errWrite
	}
	return nil
}
