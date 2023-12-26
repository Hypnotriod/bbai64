package ups

import (
	"bbai64/i2c"
	"bbai64/ina219"
	"log"
	"sync"
	"time"
)

type UpsModuleStatus struct {
	BusVoltage     float64
	CellVoltage    float64
	Current        float64 // negative current means the battery is discharging
	Power          float64
	ChargePercents float64
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
	i2c, err := i2c.Open(u.busNumber)
	if err != nil {
		log.Fatal("Could not open i2c bus ", u.busNumber)
	}
	defer i2c.Close()

	ina219 := ina219.New(i2c, ina219.ADDRESS_DEFAULT)
	if err := ina219.SetCalibration32Volts2Amps(); err != nil {
		log.Fatal("Could not initialize ina219")
	}
	for {
		busVoltage, err := ina219.ReadBusVoltage()
		if err != nil {
			log.Print("Could not update bus voltage")
		}
		current, err := ina219.ReadCurrent()
		if err != nil {
			log.Print("Could not update current")
		}
		power, err := ina219.ReadPower()
		if err != nil {
			log.Print("Could not update power")
		}

		// Assume that 4V is the maximum voltage 18650 Li-Ion battery shows under the load,
		// 4.1V is the maximum voltage 18650 Li-Ion battery can be charged to
		// and 3.5V is the minimum voltage 18650 Li-Ion battery can be discharged to
		var chargePercents float64
		if current < 0 { // Battery provides power
			chargePercents = ((busVoltage / 3) - 3.5) / 0.5 * 100
		} else { // Battery is charging
			chargePercents = ((busVoltage / 3) - 3.5) / 0.6 * 100
		}
		chargePercents = max(min(chargePercents, 0), 100)

		u.mu.Lock()
		u.status.BusVoltage = busVoltage
		u.status.CellVoltage = busVoltage / 3
		u.status.Current = current
		u.status.Power = power
		u.status.ChargePercents = chargePercents
		u.mu.Unlock()

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
