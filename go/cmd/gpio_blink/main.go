package main

import (
	"bbai64/gpio"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	led, err := gpio.Export(gpio.P8_03)
	if err != nil {
		log.Fatal("Can not export P8_03: ", err)
	}
	defer led.Unexport()

	led.SetDirection(gpio.OUT)
	defer led.SetDirection(gpio.IN)

	go func() {
		for {
			led.SetValue(gpio.HIGH)
			time.Sleep(time.Second)
			led.SetValue(gpio.LOW)
			time.Sleep(time.Second)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	led.SetValue(gpio.LOW)
}
