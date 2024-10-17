package main

import (
	"bbai64/gpio"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	led, err := gpio.Export(gpio.P8_03)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer led.Unexport()
	led.SetDirection(gpio.OUT)
	defer led.SetDirection(gpio.IN)

	button, err := gpio.Export(gpio.P8_04)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer button.Unexport()
	button.SetDirection(gpio.IN)
	button.SetEdge(gpio.RISING)
	defer button.SetEdge(gpio.NONE)

	buttonChan := make(chan struct{})
	terminateChan := make(chan os.Signal, 1)
	signal.Notify(terminateChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		button.Poll()
		buttonChan <- struct{}{}
	}()

	ledStates := [...]gpio.Value{gpio.HIGH, gpio.LOW}
	ledStateIndx := 0

	fmt.Println("Start blinking.")
blinking:
	for {
		led.SetValue(ledStates[ledStateIndx])
		ledStateIndx = (ledStateIndx + 1) % len(ledStates)
		select {
		case <-buttonChan:
			fmt.Println("Button was pressed.")
			break blinking
		case <-terminateChan:
			fmt.Println("Program was terminated.")
			break blinking
		case <-time.After(time.Second):
		}
	}
	fmt.Println("Stop blinking.")
	led.SetValue(gpio.LOW)
}
