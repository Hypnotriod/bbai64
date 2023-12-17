package main

import (
	"bbai64/command"
	"bbai64/pwm"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const SERVER_ADDRESS = ":1337"
const SERVO_PWM_PERIOD = 20 * time.Millisecond
const SERVO_PWM_DUTY_CYCLE_MIDDLE = 1500000 * time.Nanosecond
const SERVO_PWM_DUTY_CYCLE_RANGE = 400000 * time.Nanosecond

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     checkOrigin,
}

var servoSteering = pwm.NewPWM(pwm.Bus0, pwm.ChannelA)
var servoDrivetrain = pwm.NewPWM(pwm.Bus0, pwm.ChannelB)
var wsMutex sync.Mutex

func checkOrigin(r *http.Request) bool {
	return true
}

func serveWSRequest(w http.ResponseWriter, r *http.Request) {
	if !wsMutex.TryLock() {
		log.Print("Websocket multiple connections are not allowed with ", r.Host)
		return
	}
	defer wsMutex.Unlock()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Websocket upgrade error: ", err)
		return
	}
	log.Print("Websocket connection established with ", r.Host)
	defer conn.Close()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Print("Websocket read error: ", err)
			break
		}
		cmd, err := command.Unmarshal(message)
		if err != nil {
			log.Print("Websocket command format error: ", err)
			break
		}
		processCommand(cmd)
	}
	log.Print("Websocket connection terminated with ", r.Host)
}

func processCommand(cmd *command.Command) {
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
	drivetrain := normalize(values[1])
	servoSteering.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE + time.Duration(steering*float64(SERVO_PWM_DUTY_CYCLE_RANGE)))
	servoDrivetrain.DutyCycle(SERVO_PWM_DUTY_CYCLE_MIDDLE + time.Duration(drivetrain*float64(SERVO_PWM_DUTY_CYCLE_RANGE)))
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

func main() {
	initServos()

	http.HandleFunc("/ws", serveWSRequest)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.ListenAndServe(SERVER_ADDRESS, nil)
}
