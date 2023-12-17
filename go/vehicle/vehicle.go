package vehicle

import (
	"bbai64/command"
	"bbai64/pwm"
	"log"
	"time"
)

const SERVO_PWM_PERIOD = 20 * time.Millisecond
const SERVO_PWM_DUTY_CYCLE_MIDDLE = 1500000 * time.Nanosecond
const SERVO_PWM_DUTY_CYCLE_RANGE = 320000 * time.Nanosecond

var servoSteering = pwm.NewPWM(pwm.Bus0, pwm.ChannelA)
var servoDrivetrain = pwm.NewPWM(pwm.Bus0, pwm.ChannelB)

var steeringPrev float64
var drivetrainPrev float64

func Initialize() {
	initServos()
}

func Reset() {
	servoSteering.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE)
	servoDrivetrain.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE)
}

func initServos() {
	if err := servoSteering.Period(SERVO_PWM_PERIOD); err != nil {
		log.Fatal("Could not set Steering pwm period")
	}
	if err := servoSteering.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE); err != nil {
		log.Fatal("Could not set Steering pwm duty cycle")
	}
	if err := servoSteering.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Steering pwm polarity")
	}
	if err := servoSteering.Enable(); err != nil {
		log.Fatal("Could not enable Steering pwm")
	}

	if err := servoDrivetrain.Period(SERVO_PWM_PERIOD); err != nil {
		log.Fatal("Could not set Drivetrain pwm period")
	}
	if err := servoDrivetrain.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE); err != nil {
		log.Fatal("Could not set Drivetrain pwm duty cycle")
	}
	if err := servoDrivetrain.Polarity(pwm.PolarityInversed); err != nil {
		log.Fatal("Could not set Drivetrain pwm polarity")
	}
	if err := servoDrivetrain.Enable(); err != nil {
		log.Fatal("Could not enable Drivetrain pwm")
	}
}

func ProcessCommand(cmd *command.Command) {
	switch cmd.Type {
	case command.SetServoValues:
		setServoValues(cmd.Values)
	}
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
		servoSteering.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE + time.Duration(steering*float64(SERVO_PWM_DUTY_CYCLE_RANGE)))
	}
	drivetrain := normalize(values[1])
	if drivetrainPrev != drivetrain {
		drivetrainPrev = drivetrain
		servoDrivetrain.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE + time.Duration(drivetrain*float64(SERVO_PWM_DUTY_CYCLE_RANGE)))
	}
}
