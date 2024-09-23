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
	channels  map[string]chan string
	skipList  []string
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
	skipList := []string{
		"AT",
		"OK",
		"+CREG",
	}
	return &SerialSubject{
		ctx:       ctx,
		observers: make([]SerialObserver, 0),
		mu:        sync.RWMutex{},
		port:      port,
		buffer:    "",
		channels:  make(map[string]chan string),
		skipList:  skipList,
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
	timeout := 5 * time.Second
	_ = s.SendAndWaitOK("AT+CMEE=2")
	// Set the modem to text mode
	_ = s.SendAndWaitOK("AT+CMGF=1")
	// Set the modem to notify when a new SMS is received
	_ = s.SendAndWaitOK("AT+CNMI=2,2,0,0,0")
	// Enable caller ID
	_ = s.SendAndWaitOK("AT+CLIP=1")
	// Get phone service
	cops, errCops := s.SendAndGetData("+COPS", "AT+COPS?", timeout)
	if errCops != nil {
		logrus.LogrusLoggerWithContext(s.ctx).Error(errCops)
	}
	logrus.LogrusLoggerWithContext(s.ctx).Infof("COPS: %s", cops)
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
			if idx := strings.Index(s.buffer, "\r\n"); idx != -1 {
				message := s.buffer[:idx]
				s.buffer = s.buffer[idx+2:]
				//logrus.LogrusLoggerWithContext(s.ctx).Debugf("Received: %s", message)
				if message == "" {
					continue
				}
				if message == "OK" {
					s.channels["OK"] <- message
					continue
				}
				isSkip := false
				for _, skip := range s.skipList {
					if strings.HasPrefix(message, skip) {
						isSkip = true
						break
					}
				}
				if isSkip {
					continue
				}
				// check if message is response to channel
				keys := s.getAllWaitKey()
				for _, key := range keys {
					if strings.HasPrefix(message, key) {
						//logrus.LogrusLoggerWithContext(s.ctx).Debugf("Sent to %s channel: %s", key, message)
						s.channels[key] <- message
						continue
					}
				}
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

// SendAndWaitOK sends a message to the serial port and waits for a response
func (s *SerialSubject) SendAndWaitOK(command string) error {
	timeout := 5 * time.Second
	err := s.Send(command)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.channels["OK"] = make(chan string, 1)
	s.mu.Unlock()

	select {
	case <-s.channels["OK"]:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for OK response")
	}
}

func (s *SerialSubject) SendAndGetData(key string, command string, timeout time.Duration) (string, error) {
	err := s.Send(command)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.channels[key] = make(chan string, 1)
	s.mu.Unlock()

	select {
	case response := <-s.channels[key]:
		return response, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for DATA response")
	}
}

// Send sends a message to the serial port
func (s *SerialSubject) Send(command string) error {
	_, errWrite := s.port.Write([]byte(fmt.Sprintf("%s\r\n", command)))
	if errWrite != nil {
		return errWrite
	}
	return nil
}

func (s *SerialSubject) getAllWaitKey() []string {
	keys := make([]string, 0)
	s.mu.Lock()
	for k := range s.channels {
		if k == "OK" {
			continue
		}
		keys = append(keys, k)
	}
	s.mu.Unlock()
	return keys
}
