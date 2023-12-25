package main

import (
	"bbai64/i2c"
	"bbai64/ina219"
	"log"
)

func main() {
	bus, err := i2c.Open(i2c.Device1)
	if err != nil {
		log.Fatal("Can not open i2c device 1")
	}
	defer bus.Close()
	ina219, err := ina219.New(bus, ina219.ADDRESS_DEFAULT)
	if err != nil {
		log.Fatal("Can not initialize ina219")
	}
	busVoltage, err := ina219.ReadBusVoltage()
	if err != nil {
		log.Fatal("Can not read bus voltage")
	}
	current, err := ina219.ReadCurrent()
	if err != nil {
		log.Fatal("Can not read current")
	}
	log.Printf("Bus voltage: %f", busVoltage)
	log.Printf("Current: %f", current)

	percents := (busVoltage - 9) / 3.6 * 100
	if percents > 100 {
		percents = 100
	}
	if percents < 0 {
		percents = 0
	}

	log.Printf("Charge: %d%%", int(percents))
}
