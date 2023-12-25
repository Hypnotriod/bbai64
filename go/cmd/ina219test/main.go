package main

import (
	"bbai64/i2c"
	"bbai64/ina219"
	"log"
	"time"
)

func main() {
	bus, err := i2c.Open(i2c.Device1)
	if err != nil {
		log.Fatal("Can not open i2c device 1")
	}
	defer bus.Close()
	ina219 := ina219.New(bus, ina219.ADDRESS_DEFAULT)
	if err := ina219.SetCalibration32Volts2Amps(); err != nil {
		log.Fatal("Can not initialize ina219")
	}
	for {
		busVoltage, err := ina219.ReadBusVoltage()
		if err != nil {
			log.Fatal("Can not read bus voltage")
		}
		current, err := ina219.ReadCurrent()
		if err != nil {
			log.Fatal("Can not read current")
		}
		power, err := ina219.ReadPower()
		if err != nil {
			log.Fatal("Can not read power")
		}
		log.Printf("3S: %.3f V", busVoltage)
		log.Printf("1S: %.3f V", busVoltage/3)
		log.Printf("Current: %.3f A", current)
		log.Printf("Power: %.3f W", power)

		// assume that 4.1 is the maximum voltage 18650 Li-Ion battery can be charged to
		// and 3.3 is the absolute minimum voltage 18650 Li-Ion battery can be discharged to
		percents := ((busVoltage / 3) - 3.3) / 0.75 * 100
		if percents > 100 {
			percents = 100
		}
		if percents < 0 {
			percents = 0
		}

		log.Printf("Charge: %d%%", int(percents))
		log.Print("**********")

		time.Sleep(1 * time.Second)
	}
}
