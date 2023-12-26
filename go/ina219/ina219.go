package ina219

import "bbai64/i2c"

// based on: https://www.waveshare.com/wiki/UPS_Module_3S

type Register uint8

const (
	REG_CONFIG        Register = 0x00
	REG_SHUNT_VOLTAGE Register = 0x01
	REG_BUS_VOLTAGE   Register = 0x02
	REG_POWER         Register = 0x03
	REG_CURRENT       Register = 0x04
	REG_CALIBRATION   Register = 0x05
)

type BusVoltageRange uint16

const (
	RANGE_16V BusVoltageRange = 0x00 // set bus voltage range to 16V
	RANGE_32V BusVoltageRange = 0x01 // set bus voltage range to 32V (default)
)

type Gain uint16

const (
	DIV_1_40MV  Gain = 0x00 // shunt prog. gain set to  1, 40 mV range
	DIV_2_80MV  Gain = 0x01 // shunt prog. gain set to /2, 80 mV range
	DIV_4_160MV Gain = 0x02 // shunt prog. gain set to /4, 160 mV range
	DIV_8_320MV Gain = 0x03 // shunt prog. gain set to /8, 320 mV range
)

type ADCResolution uint16

const (
	ADCRES_9BIT_1S    ADCResolution = 0x00 //  9bit,   1 sample,     84us
	ADCRES_10BIT_1S   ADCResolution = 0x01 // 10bit,   1 sample,    148us
	ADCRES_11BIT_1S   ADCResolution = 0x02 // 11 bit,  1 sample,    276us
	ADCRES_12BIT_1S   ADCResolution = 0x03 // 12 bit,  1 sample,    532us
	ADCRES_12BIT_2S   ADCResolution = 0x09 // 12 bit,  2 samples,  1.06ms
	ADCRES_12BIT_4S   ADCResolution = 0x0A // 12 bit,  4 samples,  2.13ms
	ADCRES_12BIT_8S   ADCResolution = 0x0B // 12bit,   8 samples,  4.26ms
	ADCRES_12BIT_16S  ADCResolution = 0x0C // 12bit,  16 samples,  8.51ms
	ADCRES_12BIT_32S  ADCResolution = 0x0D // 12bit,  32 samples, 17.02ms
	ADCRES_12BIT_64S  ADCResolution = 0x0E // 12bit,  64 samples, 34.05ms
	ADCRES_12BIT_128S ADCResolution = 0x0F // 12bit, 128 samples, 68.10ms
)

type Mode uint16

const (
	POWERDOW             Mode = 0x00 // power down
	SVOLT_TRIGGERED      Mode = 0x01 // shunt voltage triggered
	BVOLT_TRIGGERED      Mode = 0x02 // bus voltage triggered
	SANDBVOLT_TRIGGERED  Mode = 0x03 // shunt and bus voltage triggered
	ADC_OFF              Mode = 0x04 // ADC off
	SVOLT_CONTINUOUS     Mode = 0x05 // shunt voltage continuous
	BVOLT_CONTINUOUS     Mode = 0x06 // bus voltage continuous
	SANDBVOLT_CONTINUOUS Mode = 0x07 // shunt and bus voltage continuous
)

const ADDRESS_DEFAULT uint8 = 0x41

type INA219 struct {
	bus        *i2c.Bus
	address    uint8
	currentLSB float64
	powerLSB   float64
}

func New(bus *i2c.Bus, address uint8) *INA219 {
	return &INA219{
		bus:     bus,
		address: address,
	}
}

// Configures to INA219 to be able to measure up to 32V and 2A of current.
// Counter overflow occurs at 3.2A.
// These calculations assume a 0.1 shunt ohm resistor is present
func (i *INA219) SetCalibration32Volts2Amps() error {
	i.currentLSB = 0.1 // Current LSB = 100uA per bit
	i.powerLSB = 2     // Power LSB = 2mW per bit
	var calibrationValue uint16 = 4096
	if err := i.bus.WriteWord(i.address, uint8(REG_CALIBRATION), calibrationValue); err != nil {
		return err
	}
	return i.writeConfig(RANGE_32V, DIV_8_320MV, ADCRES_12BIT_32S, ADCRES_12BIT_32S, SANDBVOLT_CONTINUOUS)
}

func (i *INA219) writeConfig(
	busVoltageRange BusVoltageRange,
	gain Gain,
	busADCResolution ADCResolution,
	shuntADCResolution ADCResolution,
	mode Mode) error {
	config := uint16(busVoltageRange<<13) |
		uint16(gain<<11) |
		uint16(busADCResolution<<7) |
		uint16(shuntADCResolution<<3) |
		uint16(mode)
	return i.bus.WriteWord(i.address, uint8(REG_CONFIG), config)
}

func (i *INA219) ReadShuntVoltage() (float64, error) {
	value, err := i.bus.ReadWord(i.address, uint8(REG_SHUNT_VOLTAGE))
	if err != nil {
		return 0, err
	}
	result := float64(int16(value)) * 0.01 / 1000
	return result, nil
}

func (i *INA219) ReadBusVoltage() (float64, error) {
	value, err := i.bus.ReadWord(i.address, uint8(REG_BUS_VOLTAGE))
	if err != nil {
		return 0, err
	}
	result := float64((value >> 3)) * 4 / 1000
	return result, nil
}

func (i *INA219) ReadCurrent() (float64, error) {
	value, err := i.bus.ReadWord(i.address, uint8(REG_CURRENT))
	if err != nil {
		return 0, err
	}
	result := float64(int16(value)) * i.currentLSB / 1000
	return result, nil
}

func (i *INA219) ReadPower() (float64, error) {
	value, err := i.bus.ReadWord(i.address, uint8(REG_POWER))
	if err != nil {
		return 0, err
	}
	result := float64(int16(value)) * i.powerLSB / 1000
	return result, nil
}
