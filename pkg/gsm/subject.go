package gsm

import (
	"context"
	"fmt"
	"go-gsm/pkg/logrus"
	"go.bug.st/serial"
	"strings"
	"sync"
	"time"
)

type SerialObserver interface {
	Update(data string)
}

type SerialSubject struct {
	ctx       *context.Context
	observers []SerialObserver
	mu        sync.RWMutex
	port      serial.Port
	buffer    string
	phone     string
	ccid      string
	signal    int
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

func NewSerial(ctx *context.Context, port serial.Port) *SerialSubject {
	return &SerialSubject{
		ctx:       ctx,
		observers: make([]SerialObserver, 0),
		mu:        sync.RWMutex{},
		port:      port,
		buffer:    "",
	}
}

// attach adds an observer to the list of observers
func (s *SerialSubject) attach(observer SerialObserver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, observer)
}

// notify sends a message to all observers
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
	s.attach(NewCallObserver(s))
	s.attach(NewInfoObserver(s))
	go s.read()
	// Enable error messages
	time.Sleep(1 * time.Second)
	if err := s.Send("AT+CMEE=2"); err != nil {
		return err
	}
	// Set the modem to text mode
	time.Sleep(1 * time.Second)
	if err := s.Send("AT+CMGF=1"); err != nil {
		return err
	}
	// Set the modem to notify when a new SMS is received
	time.Sleep(1 * time.Second)
	if err := s.Send("AT+CNMI=2,2,0,0,0"); err != nil {
		return err
	}
	// Enable caller ID
	time.Sleep(1 * time.Second)
	if err := s.Send("AT+CLIP=1"); err != nil {
		return err
	}
	// Send the USSD code
	time.Sleep(1 * time.Second)
	if err := s.Send("AT+CUSD=1,\"*101#\",15"); err != nil {
		return err
	}
	// Set the modem to auto network selection
	time.Sleep(1 * time.Second)
	if err := s.Send("AT+QCFG=\"nwscanmode\",0,1"); err != nil {
		return err
	}
	// Get the SIM card number
	time.Sleep(1 * time.Second)
	if err := s.Send("AT+CCID"); err != nil {
		return err
	}
	go s.getNetworkSignal()
	return nil
}

func (s *SerialSubject) getNetworkSignal() {
	for {
		if err := s.Send("AT+CSQ"); err != nil {
			logrus.LogrusLoggerWithContext(s.ctx).Errorf("Error getting network signal: %v", err)
		}
		time.Sleep(30 * time.Second)
	}
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
func (s *SerialSubject) Send(command string) error {
	logrus.LogrusLoggerWithContext(s.ctx).Infof("Sending: %s", command)
	_, errWrite := s.port.Write([]byte(fmt.Sprintf("%s\r\n", command)))
	if errWrite != nil {
		return errWrite
	}
	return nil
}
