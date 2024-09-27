package gsm

import (
	"context"
	"fmt"
	"go-gsm/pkg/logrus"
	"go.bug.st/serial"
	"os"
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
	portName  string
	buffer    string
	phone     string
	network   string
	ccid      string
	signal    int
	channels  map[string]chan string
	skipList  []string
	cusd      string
	wavBuffer []byte
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

func NewSerial(ctx *context.Context, port serial.Port, portName string) *SerialSubject {
	skipList := []string{
		"AT",
		"OK",
		"+CREG",
		"CONNECT",
	}
	return &SerialSubject{
		ctx:       ctx,
		observers: make([]SerialObserver, 0),
		mu:        sync.RWMutex{},
		port:      port,
		portName:  portName,
		buffer:    "",
		channels:  make(map[string]chan string),
		skipList:  skipList,
		cusd:      "",
		wavBuffer: nil,
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
	s.attach(NewSMSObserver(s))
	s.attach(NewCallObserver(s))
	s.attach(NewInfoObserver(s))
	go s.read()
	// Enable error messages
	_ = s.SendAndWaitOK("AT+CMEE=2")
	// Set the modem to text mode
	_ = s.SendAndWaitOK("AT+CMGF=1")
	// Set the modem to notify when a new SMS is received
	_ = s.SendAndWaitOK("AT+CNMI=2,2,0,0,0")
	// Enable caller ID
	_ = s.SendAndWaitOK("AT+CLIP=1")
	// Delete all files in the file system
	//_ = s.SendAndWaitOK("AT+QFDEL=\"*\"")
	// Get ICCID
	ccid, errCCID := s.SendAndGetData("+CCID", "AT+CCID", 5*time.Second)
	if errCCID != nil {
		logrus.LogrusLoggerWithContext(s.ctx).Error(errCCID)
	}
	s.ccid = ccid
	// Get phone service
	cops, errCops := s.SendAndGetData("+COPS", "AT+COPS?", 5*time.Second)
	if errCops != nil {
		logrus.LogrusLoggerWithContext(s.ctx).Error(errCops)
	}
	network := extractNetwork(cops)
	logrus.LogrusLoggerWithContext(s.ctx).Infof("COPS: %s", cops)
	logrus.LogrusLoggerWithContext(s.ctx).Infof("Network: %s", network)
	ussd := "*101#"
	if network == "Vinaphone" {
		ussd = "*111#"
	}
	cusd, errCUSD := s.SendUSSD(ussd)
	if errCUSD != nil {
		logrus.LogrusLoggerWithContext(s.ctx).Error(errCUSD)
	}
	logrus.LogrusLoggerWithContext(s.ctx).Infof("CUSD: %s", cusd)
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
				if message == "" {
					continue
				}
				if s.wavBuffer == nil && strings.HasPrefix(message, "RIFF") {
					// Nếu chưa có buffer và thông điệp bắt đầu bằng "RIFF"
					i := strings.Index(message, "RIFF")
					s.wavBuffer = []byte(message[i:])
					continue
				}

				if s.wavBuffer != nil {
					// Nếu đã có buffer, kiểm tra chuỗi "+QFDWL"
					i := strings.Index(message, "+QFDWL")
					if i != -1 {
						// Nối phần tin nhắn có "+QFDWL" vào buffer
						s.wavBuffer = append(s.wavBuffer, []byte(message[i:])...)
						logrus.LogrusLoggerWithContext(s.ctx).Info("Received WAV file")

						// Ghi file WAV
						if errWrite := os.WriteFile(fmt.Sprintf("%s.wav", s.portName), s.wavBuffer, 0644); errWrite != nil {
							logrus.LogrusLoggerWithContext(s.ctx).Errorf("Error writing WAV file: %v", errWrite)
						}

						// Xóa buffer sau khi ghi xong
						s.wavBuffer = nil
						continue
					} else {
						// Nếu không tìm thấy "+QFDWL", tiếp tục nối dữ liệu vào buffer
						s.wavBuffer = append(s.wavBuffer, []byte(message)...)
						continue
					}
				}

				if message == "OK" {
					if _, ok := s.channels["OK"]; ok {
						s.channels["OK"] <- message
					}
					continue
				}
				logrus.LogrusLoggerWithContext(s.ctx).Debugf("Received: %s", message)
				isSkip := false
				for _, skip := range s.skipList {
					if strings.HasPrefix(message, skip) {
						isSkip = true
						break
					}
				}
				if isSkip {
					logrus.LogrusLoggerWithContext(s.ctx).Debugf("Skipped: %s", message)
					continue
				}
				// check if message is response to channel
				keys := s.getWaitKeys()
				for _, key := range keys {
					if strings.HasPrefix(message, key) {
						//logrus.LogrusLoggerWithContext(s.ctx).Debugf("Sent to %s channel: %s", key, message)
						s.channels[key] <- message
						continue
					}
				}
				if strings.HasPrefix(message, "+CUSD:") {
					firstIndex := strings.Index(message, "\"")
					lastIndex := strings.LastIndex(message, "\"")
					if firstIndex == lastIndex {
						s.cusd = message[firstIndex+1:] + "\n"
						continue
					} else {
						s.cusd = message[firstIndex+1:lastIndex] + "\n"
						s.channels["USSD"] <- strings.TrimSpace(s.cusd)
						s.cusd = ""
						continue
					}
				}
				if s.cusd != "" {
					index := strings.Index(message, "\"")
					if index != -1 {
						s.cusd += message[:index]
						s.channels["USSD"] <- strings.TrimSpace(s.cusd)
						s.cusd = ""
						continue
					}
					s.cusd += message + "\n"
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
	err := s.Send(command)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.channels["OK"] = make(chan string, 1)
	s.mu.Unlock()

	select {
	case <-s.channels["OK"]:
		delete(s.channels, "OK")
		return nil
	case <-time.After(1 * time.Second):
		delete(s.channels, "OK")
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
	logrus.LogrusLoggerWithContext(s.ctx).Warnf("Sending: %s", command)
	_, errWrite := s.port.Write([]byte(fmt.Sprintf("%s\r\n", command)))
	if errWrite != nil {
		return errWrite
	}
	return nil
}

func (s *SerialSubject) SendUSSD(ussd string) (string, error) {
	err := s.Send(fmt.Sprintf("AT+CUSD=1,\"%s\",15", ussd))
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.channels["USSD"] = make(chan string, 1)
	s.mu.Unlock()
	time.Sleep(1 * time.Second)

	select {
	case response := <-s.channels["USSD"]:
		return response, nil
	case <-time.After(60 * time.Second):
		return "", fmt.Errorf("timeout waiting for USSD response")
	}
}

func (s *SerialSubject) getWaitKeys() []string {
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
