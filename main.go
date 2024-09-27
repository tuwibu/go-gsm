package main

import (
	"context"
	"fmt"
	"go-gsm/pkg/gsm"
	"go-gsm/pkg/logrus"
)

func main() {
	ctx := context.Background()
	logrus.InitLogrusLogger()
	ports, errPorts := gsm.GetAvailablePorts()
	if errPorts != nil {
		panic(errPorts)
	}
	logrus.LogrusLoggerWithContext(&ctx).Debugf("Available ports: %v", ports)
	// COM32, COM21, COM20
	portName := "COM32"
	port, errPort := gsm.CreatePort(portName, 115200)
	if errPort != nil {
		panic(errPort)
	}
	serial := gsm.NewSerial(&ctx, port, "ccc")
	if err := serial.Open(); err != nil {
		panic(err)
	}
	//if err := serial.Send("AT+QFLST"); err != nil {
	//	panic(err)
	//}
	if err := serial.Send(fmt.Sprintf("AT+QFDWL=\"%s.wav\";\r", "ccc")); err != nil {
		panic(err)
	}
	forever := make(chan struct{})
	logrus.LogrusLoggerWithContext(&ctx).Info("Press Ctrl+C to exit")
	<-forever
}
