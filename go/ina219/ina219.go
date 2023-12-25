package ina219

import "bbai64/i2c"

// based on: https://www.waveshare.com/wiki/UPS_Module_3S

const (
	// Config Register (R/W)
	_REG_CONFIG uint8 = 0x00
	// SHUNT VOLTAGE REGISTER (R)
	_REG_SHUNTVOLTAGE uint8 = 0x01

	// BUS VOLTAGE REGISTER (R)
	_REG_BUSVOLTAGE uint8 = 0x02

	// POWER REGISTER (R)
	_REG_POWER uint8 = 0x03

	// CURRENT REGISTER (R)
	_REG_CURRENT uint8 = 0x04

	// CALIBRATION REGISTER (R/W)
	_REG_CALIBRATION uint8 = 0x05
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
	ADCOFF               Mode = 0x04 // ADC off
	SVOLT_CONTINUOUS     Mode = 0x05 // shunt voltage continuous
	BVOLT_CONTINUOUS     Mode = 0x06 // bus voltage continuous
	SANDBVOLT_CONTINUOUS Mode = 0x07 // shunt and bus voltage continuous
)

const ADDRESS_DEFAULT uint8 = 0x41

type INA219 struct {
	bus                  *i2c.Bus
	address              uint8
	bus_voltage_range    BusVoltageRange
	gain                 Gain
	bus_adc_resolution   ADCResolution
	shunt_adc_resolution ADCResolution
	mode                 Mode
	config               uint16
	_cal_value           uint16
	_current_lsb         float32
	_power_lsb           float32
}

func New(bus *i2c.Bus, address uint8) (*INA219, error) {
	ina219 := &INA219{
		bus:     bus,
		address: address,
	}
	err := ina219.setCalibration32Volts2Amps()
	return ina219, err
}

// Configures to INA219 to be able to measure up to 32V and 2A of current.
// Counter overflow occurs at 3.2A.
// These calculations assume a 0.1 shunt ohm resistor is present
func (i *INA219) setCalibration32Volts2Amps() error {
	// By default we use a pretty huge range for the input voltage,
	// which probably isn't the most appropriate choice for system
	// that don't use a lot of power.  But all of the calculations
	// are shown below if you want to change the settings.  You will
	// also need to change any relevant register settings, such as
	// setting the VBUS_MAX to 16V instead of 32V, etc.

	// VBUS_MAX = 32V             (Assumes 32V, can also be set to 16V)
	// VSHUNT_MAX = 0.32          (Assumes Gain 8, 320mV, can also be 0.16, 0.08, 0.04)
	// RSHUNT = 0.1               (Resistor value in ohms)

	// 1. Determine max possible current
	// MaxPossible_I = VSHUNT_MAX / RSHUNT
	// MaxPossible_I = 3.2A

	// 2. Determine max expected current
	// MaxExpected_I = 2.0A

	// 3. Calculate possible range of LSBs (Min = 15-bit, Max = 12-bit)
	// MinimumLSB = MaxExpected_I/32767
	// MinimumLSB = 0.000061              (61uA per bit)
	// MaximumLSB = MaxExpected_I/4096
	// MaximumLSB = 0,000488              (488uA per bit)

	// 4. Choose an LSB between the min and max values
	//    (Preferrably a roundish number close to MinLSB)
	// CurrentLSB = 0.0001 (100uA per bit)
	i._current_lsb = 0.1 // Current LSB = 100uA per bit

	// 5. Compute the calibration register
	// Cal = trunc (0.04096 / (Current_LSB * RSHUNT))
	// Cal = 4096 (0x1000)

	i._cal_value = 4096

	// 6. Calculate the power LSB
	// PowerLSB = 20 * CurrentLSB
	// PowerLSB = 0.002 (2mW per bit)
	i._power_lsb = 0.002 // Power LSB = 2mW per bit

	// 7. Compute the maximum current and shunt voltage values before overflow
	//
	// Max_Current = Current_LSB * 32767
	// Max_Current = 3.2767A before overflow
	//
	// If Max_Current > Max_Possible_I then
	//    Max_Current_Before_Overflow = MaxPossible_I
	// Else
	//    Max_Current_Before_Overflow = Max_Current
	// End If
	//
	// Max_ShuntVoltage = Max_Current_Before_Overflow * RSHUNT
	// Max_ShuntVoltage = 0.32V
	//
	// If Max_ShuntVoltage >= VSHUNT_MAX
	//    Max_ShuntVoltage_Before_Overflow = VSHUNT_MAX
	// Else
	//    Max_ShuntVoltage_Before_Overflow = Max_ShuntVoltage
	// End If

	// 8. Compute the Maximum Power
	// MaximumPower = Max_Current_Before_Overflow * VBUS_MAX
	// MaximumPower = 3.2 * 32V
	// MaximumPower = 102.4W

	// Set Calibration register to 'Cal' calculated above
	if err := i.bus.WriteWord(i.address, _REG_CALIBRATION, i._cal_value); err != nil {
		return err
	}

	// Set Config register to take into account the settings above
	i.bus_voltage_range = RANGE_32V
	i.gain = DIV_8_320MV
	i.bus_adc_resolution = ADCRES_12BIT_32S
	i.shunt_adc_resolution = ADCRES_12BIT_32S
	i.mode = SANDBVOLT_CONTINUOUS
	i.config = uint16(i.bus_voltage_range<<13) |
		uint16(i.gain<<11) |
		uint16(i.bus_adc_resolution<<7) |
		uint16(i.shunt_adc_resolution<<3) |
		uint16(i.mode)
	return i.bus.WriteWord(i.address, _REG_CONFIG, i.config)
}

func (i *INA219) ReadShuntVoltage() (float32, error) {
	if err := i.bus.WriteWord(i.address, _REG_CALIBRATION, i._cal_value); err != nil {
		return 0, err
	}
	value, err := i.bus.ReadWord(i.address, _REG_SHUNTVOLTAGE)
	if err != nil {
		return 0, err
	}
	if value > 32767 {
		value -= 65535
	}
	return float32(value) * 0.00001, nil
}

func (i *INA219) ReadBusVoltage() (float32, error) {
	if err := i.bus.WriteWord(i.address, _REG_CALIBRATION, i._cal_value); err != nil {
		return 0, err
	}
	i.bus.ReadWord(i.address, _REG_BUSVOLTAGE)
	value, err := i.bus.ReadWord(i.address, _REG_BUSVOLTAGE)
	if err != nil {
		return 0, err
	}
	return float32((value >> 3)) * 0.004, nil
}

func (i *INA219) ReadCurrent() (float32, error) {
	value, err := i.bus.ReadWord(i.address, _REG_CURRENT)
	if err != nil {
		return 0, err
	}
	if value > 32767 {
		value -= 65535
	}
	return float32(value) * i._current_lsb * 0.001, nil
}

func (i *INA219) ReadPower() (float32, error) {
	value, err := i.bus.ReadWord(i.address, _REG_POWER)
	if err != nil {
		return 0, err
	}
	if value > 32767 {
		value -= 65535
	}
	return float32(value) * i._current_lsb, nil
}
