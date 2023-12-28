package vehicle

import (
	"bbai64/pwm"
	"log"
	"time"
)

type State struct {
	ServoValues []float64 `json:"servoValues"`
}

const PWM_PERIOD = 20 * time.Millisecond
const PWM_DUTY_CYCLE_MIDDLE = 1500000 * time.Nanosecond
const SERVO_PWM_DUTY_CYCLE_RANGE = 320000 * time.Nanosecond

var servoSteering = pwm.NewPWM(pwm.Bus0, pwm.ChannelA)
var servoThrottle = pwm.NewPWM(pwm.Bus0, pwm.ChannelB)

var steeringPrev float64
var throttlePrev float64

func Initialize() {
	initServos()
}

func Reset() {
	servoSteering.DutyCycle(PWM_DUTY_CYCLE_MIDDLE)
	servoThrottle.DutyCycle(PWM_DUTY_CYCLE_MIDDLE)
}

func initServos() {
	if err := servoSteering.Period(PWM_PERIOD); err != nil {
		log.Fatal("Could not set Steering pwm period")
	}
	if err := servoSteering.DutyCycle(PWM_DUTY_CYCLE_MIDDLE); err != nil {
		log.Fatal("Could not set Steering pwm duty cycle")
	}
	if err := servoSteering.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Steering pwm polarity")
	}
	if err := servoSteering.Enable(); err != nil {
		log.Fatal("Could not enable Steering pwm")
	}

	if err := servoThrottle.Period(PWM_PERIOD); err != nil {
		log.Fatal("Could not set Throttle pwm period")
	}
	if err := servoThrottle.DutyCycle(PWM_DUTY_CYCLE_MIDDLE); err != nil {
		log.Fatal("Could not set Throttle pwm duty cycle")
	}
	if err := servoThrottle.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Throttle pwm polarity")
	}
	if err := servoThrottle.Enable(); err != nil {
		log.Fatal("Could not enable Throttle pwm")
	}
}

func UpdateWithState(status *State) {
	setServoValues(status.ServoValues)
}

func normalize(value float64) float64 {
	if value < -1 {
		return -1
	}
	if value > 1 {
		return 1
	}
	return value
}

func setServoValues(values []float64) {
	if len(values) != 2 {
		log.Print("Servo values are invalid")
		return
	}
	steering := normalize(values[0])
	if steeringPrev != steering {
		steeringPrev = steering
		servoSteering.DutyCycle(PWM_DUTY_CYCLE_MIDDLE + time.Duration(steering*float64(SERVO_PWM_DUTY_CYCLE_RANGE)))
	}
	throttle := normalize(values[1])
	if throttlePrev != throttle {
		throttlePrev = throttle
		servoThrottle.DutyCycle(PWM_DUTY_CYCLE_MIDDLE + time.Duration(throttle*float64(SERVO_PWM_DUTY_CYCLE_RANGE)))
	}
}
