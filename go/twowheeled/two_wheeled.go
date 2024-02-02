package twowheeled

import (
	"bbai64/pwm"
	"log"
	"time"
)

type State struct {
	Inputs []float64 `json:"inputs"`
}

const PWM_PERIOD = 1 * time.Millisecond
const PWM_DUTY_CYCLE_MAX = 400000 * time.Nanosecond // cap to 40% of max power

var wheelLeftForward = pwm.NewPWM(pwm.Bus0, pwm.ChannelA)
var wheelLeftBackward = pwm.NewPWM(pwm.Bus0, pwm.ChannelB)
var wheelRightForward = pwm.NewPWM(pwm.Bus1, pwm.ChannelA)
var wheelRightBackward = pwm.NewPWM(pwm.Bus1, pwm.ChannelB)

var leftSpeedPrev float64
var rightSpeedPrev float64

func Initialize() {
	initWheels()
}

func Reset() {
	wheelLeftForward.DutyCycle(0)
	wheelLeftBackward.DutyCycle(0)
	wheelRightForward.DutyCycle(0)
	wheelRightBackward.DutyCycle(0)
}

func initWheels() {
	if err := wheelLeftForward.Period(PWM_PERIOD); err != nil {
		log.Fatal("Could not set Left Wheel Forward pwm period")
	}
	if err := wheelLeftForward.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Left Wheel Forward pwm polarity")
	}
	if err := wheelLeftForward.Enable(); err != nil {
		log.Fatal("Could not enable Left Wheel Forward pwm")
	}
	if err := wheelLeftForward.DutyCycle(0); err != nil {
		log.Fatal("Could not set Left Wheel Forward pwm duty cycle")
	}

	if err := wheelLeftBackward.Period(PWM_PERIOD); err != nil {
		log.Fatal("Could not set Left Wheel Backward pwm period")
	}
	if err := wheelLeftBackward.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Left Wheel Backward pwm polarity")
	}
	if err := wheelLeftBackward.Enable(); err != nil {
		log.Fatal("Could not enable Left Wheel Backward pwm")
	}
	if err := wheelLeftBackward.DutyCycle(0); err != nil {
		log.Fatal("Could not set Left Wheel Backward pwm duty cycle")
	}

	if err := wheelRightForward.Period(PWM_PERIOD); err != nil {
		log.Fatal("Could not set Right Wheel Forward pwm period")
	}
	if err := wheelRightForward.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Right Wheel Forward pwm polarity")
	}
	if err := wheelRightForward.Enable(); err != nil {
		log.Fatal("Could not enable Right Wheel Forward pwm")
	}
	if err := wheelRightForward.DutyCycle(0); err != nil {
		log.Fatal("Could not set Right Wheel Forward pwm duty cycle")
	}

	if err := wheelRightBackward.Period(PWM_PERIOD); err != nil {
		log.Fatal("Could not set Right Wheel Backward pwm period")
	}
	if err := wheelRightBackward.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Right Wheel Backward pwm polarity")
	}
	if err := wheelRightBackward.Enable(); err != nil {
		log.Fatal("Could not enable Right Wheel Backward pwm")
	}
	if err := wheelRightBackward.DutyCycle(0); err != nil {
		log.Fatal("Could not set Right Wheel Backward pwm duty cycle")
	}
}

func UpdateWithState(status *State) {
	setWheelsValues(status.Inputs)
}

func setWheelsValues(values []float64) {
	if len(values) != 2 {
		log.Print("Control values are invalid")
		return
	}
	steering := min(max(values[0], -1), 1)
	throttle := min(max(values[1], -1), 1)
	leftSpeed := min(max(-steering+throttle, -1), 1)
	rightSpeed := min(max(steering+throttle, -1), 1)

	if leftSpeedPrev != leftSpeed {
		leftSpeedPrev = leftSpeed
		if leftSpeed >= 0 {
			wheelLeftBackward.DutyCycle(0)
			wheelLeftForward.DutyCycle(PWM_DUTY_CYCLE_MAX * time.Duration(leftSpeed))
		} else {
			wheelLeftForward.DutyCycle(0)
			wheelLeftBackward.DutyCycle(PWM_DUTY_CYCLE_MAX * time.Duration(-leftSpeed))
		}
	}

	if rightSpeedPrev != rightSpeed {
		rightSpeedPrev = rightSpeed
		if leftSpeed >= 0 {
			wheelRightBackward.DutyCycle(0)
			wheelRightForward.DutyCycle(PWM_DUTY_CYCLE_MAX * time.Duration(rightSpeed))
		} else {
			wheelRightForward.DutyCycle(0)
			wheelRightBackward.DutyCycle(PWM_DUTY_CYCLE_MAX * time.Duration(-rightSpeed))
		}
	}
}
