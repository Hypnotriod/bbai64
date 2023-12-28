package ups

import (
	"bbai64/i2c"
	"bbai64/ina219"
	"log"
	"sync"
	"time"
)

// negative ShuntVolate and Current means the battery is discharging
type UpsModuleStatus struct {
	BusVoltage     float64 `json:"busVoltage"`
	ShuntVolate    float64 `json:"shuntVolate"`
	BatteryVoltage float64 `json:"batteryVoltage"`
	CellVoltage    float64 `json:"cellVoltage"`
	Current        float64 `json:"current"`
	Power          float64 `json:"power"`
	ChargePercents float64 `json:"chargePercents"`
}

type UpsModule3S struct {
	mu        sync.RWMutex
	busNumber i2c.BusNumber
	status    UpsModuleStatus
	stop      chan bool
}

func NewUpsModule3S(busNumber i2c.BusNumber) *UpsModule3S {
	return &UpsModule3S{
		busNumber: busNumber,
	}
}

func (u *UpsModule3S) Run(refreshPeriod time.Duration) {
	bus, err := i2c.Open(u.busNumber)
	if err != nil {
		log.Fatal("Could not open i2c bus ", u.busNumber)
	}
	defer bus.Close()

	ina219 := ina219.New(bus, ina219.ADDRESS_DEFAULT)
	if err := ina219.SetCalibration32Volts2Amps(); err != nil {
		log.Fatal("Could not initialize ina219")
	}

	var busVoltage float64
	var shuntVoltage float64
	var batteryVoltage float64
	var current float64
	var power float64
	var chargePercents float64
	for {
		shuntVoltage, err = ina219.ReadShuntVoltage()
		if err != nil {
			log.Fatal("Failed to read shunt voltage")
			goto skip
		}
		busVoltage, err = ina219.ReadBusVoltage()
		if err != nil {
			log.Print("Failed to read bus voltage")
			goto skip
		}
		current, err = ina219.ReadCurrent()
		if err != nil {
			log.Print("Failed to read current")
			goto skip
		}
		power, err = ina219.ReadPower()
		if err != nil {
			log.Print("Failed to read power")
			goto skip
		}
		batteryVoltage = busVoltage - shuntVoltage
		// Assume that 4.0V is the maximum voltage 18650 Li-Ion battery can be charged to
		// and 3.5V is the minimum voltage 18650 Li-Ion battery can be discharged to
		chargePercents = ((batteryVoltage / 3) - 3.5) / 0.5 * 100
		chargePercents = max(min(chargePercents, 0), 100)

		u.mu.Lock()
		u.status.BusVoltage = busVoltage
		u.status.ShuntVolate = shuntVoltage
		u.status.BatteryVoltage = batteryVoltage
		u.status.CellVoltage = batteryVoltage / 3
		u.status.Current = current
		u.status.Power = power
		u.status.ChargePercents = chargePercents
		u.mu.Unlock()

	skip:
		select {
		case <-u.stop:
			break
		case <-time.After(refreshPeriod):
		}
	}
}

func (u *UpsModule3S) Stop() {
	u.stop <- true
}

func (u *UpsModule3S) Status() UpsModuleStatus {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.status
}
