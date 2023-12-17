package pwm

import (
	"fmt"
	"os"
	"time"
)

type Bus int

const (
	Bus0 Bus = 0
	Bus1 Bus = 1
	Bus2 Bus = 2
)

type Channel string

const (
	ChannelA Channel = "a"
	ChannelB Channel = "b"
)

type Polarity string

const (
	PolarityNormal   Polarity = "normal"
	PolarityInversed Polarity = "inversed"
)

type PWM struct {
	enable    string
	dutyCycle string
	period    string
	polarity  string
}

func NewPWM(bus Bus, channel Channel) *PWM {
	return &PWM{
		enable:    fmt.Sprintf("/dev/bone/pwm/%d/%s/enable", bus, channel),
		dutyCycle: fmt.Sprintf("/dev/bone/pwm/%d/%s/duty_cycle", bus, channel),
		period:    fmt.Sprintf("/dev/bone/pwm/%d/%s/period", bus, channel),
		polarity:  fmt.Sprintf("/dev/bone/pwm/%d/%s/polarity", bus, channel),
	}
}

func (pwm *PWM) Enable() error {
	return os.WriteFile(pwm.enable, []byte{'1'}, 0666)
}

func (pwm *PWM) Disable() error {
	return os.WriteFile(pwm.enable, []byte{'0'}, 0666)
}

func (pwm *PWM) Polarity(polarity Polarity) error {
	return os.WriteFile(pwm.polarity, []byte(polarity), 0666)
}

func (pwm *PWM) Period(period time.Duration) error {
	value := fmt.Sprintf("%d", period.Nanoseconds())
	return os.WriteFile(pwm.period, []byte(value), 0666)
}

func (pwm *PWM) DutyCycle(dutyCycle time.Duration) error {
	value := fmt.Sprintf("%d", dutyCycle.Nanoseconds())
	return os.WriteFile(pwm.dutyCycle, []byte(value), 0666)
}
