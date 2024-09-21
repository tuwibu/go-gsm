package main

import (
	"go-gsm/pkg/gsm"
	"log"
	"time"
)

func main() {
	ports, errPorts := gsm.GetAvailablePorts()
	if errPorts != nil {
		panic(errPorts)
	}
	log.Println("Available ports:", ports)
	port, errPort := gsm.CreatePort("COM32", 115200)
	if errPort != nil {
		panic(errPort)
	}
	serial := gsm.NewSerial(port)
	if err := serial.Open(); err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)
	if err := serial.Send("AT+CUSD=1,\"*101#\",15"); err != nil {
		log.Fatalln("Error sending USSD:", err)
		//panic(err)
	}
	forever := make(chan struct{})
	log.Println("Press Ctrl+C to exit")
	<-forever
}
