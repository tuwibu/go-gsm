package main

import (
	"context"
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
	// COM32, COM21
	port, errPort := gsm.CreatePort("COM32", 115200)
	if errPort != nil {
		panic(errPort)
	}
	serial := gsm.NewSerial(&ctx, port)
	if err := serial.Open(); err != nil {
		panic(err)
	}
	forever := make(chan struct{})
	logrus.LogrusLoggerWithContext(&ctx).Info("Press Ctrl+C to exit")
	<-forever
}
