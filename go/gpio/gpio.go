package gpio

/*
#ifndef GPIO_HELPER_H_
#define GPIO_HELPER_H_

#cgo CFLAGS: -std=c99
#cgo CXXFLAGS: -std=c99

#include <poll.h>

void GpioPoll(int fd)
{
	struct pollfd pfd;
	pfd.fd = fd;
	pfd.events = POLLPRI | POLLERR;
	pfd.revents = 0;
	poll(&pfd, 1, -1);
}

#endif // GPIO_HELPER_H_
*/
import "C"
import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Alias string

const (
	P8_03 Alias = "P8_03"
	P8_04 Alias = "P8_04 (BOOTMODE2)"
	P8_05 Alias = "P8_05"
	P8_06 Alias = "P8_06"
	P8_07 Alias = "P8_07"
	P8_08 Alias = "P8_08"
	P8_09 Alias = "P8_09"
	P8_10 Alias = "P8_10"
	P8_11 Alias = "P8_11 (BOOTMODE7)"
	P8_12 Alias = "P8_12"
	P8_13 Alias = "P8_13"
	P8_14 Alias = "P8_14"
	P8_15 Alias = "P8_15"
	P8_16 Alias = "P8_16"
	P8_17 Alias = "P8_17"
	P8_18 Alias = "P8_18"
	P8_19 Alias = "P8_19"
	P8_20 Alias = "P8_20"
	P8_21 Alias = "P8_21"
	P8_22 Alias = "P8_22"
	P8_23 Alias = "P8_23"
	P8_24 Alias = "P8_24"
	P8_25 Alias = "P8_25"
	P8_26 Alias = "P8_26"
	P8_27 Alias = "P8_27"
	P8_28 Alias = "P8_28"
	P8_29 Alias = "P8_29"
	P8_30 Alias = "P8_30"
	P8_31 Alias = "P8_31A"
	P8_32 Alias = "P8_32A"
	P8_33 Alias = "P8_33A"
	P8_34 Alias = "P8_34"
	P8_35 Alias = "P8_35A"
	P8_36 Alias = "P8_36"
	P8_37 Alias = "P8_37A"
	P8_38 Alias = "P8_38A"
	P8_39 Alias = "P8_39"
	P8_40 Alias = "P8_40"
	P8_41 Alias = "P8_41"
	P8_42 Alias = "P8_42 (BOOTMODE6)"
	P8_43 Alias = "P8_43"
	P8_44 Alias = "P8_44"
	P8_45 Alias = "P8_45"
	P8_46 Alias = "P8_46 (BOOTMODE3)"
	P9_11 Alias = "P9_11"
	P9_12 Alias = "P9_12"
	P9_13 Alias = "P9_13"
	P9_14 Alias = "P9_14"
	P9_15 Alias = "P9_15"
	P9_16 Alias = "P9_16"
	P9_17 Alias = "P9_17A"
	P9_18 Alias = "P9_18A"
	P9_19 Alias = "P9_19A"
	P9_20 Alias = "P9_20A"
	P9_21 Alias = "P9_21A"
	P9_22 Alias = "P9_22A (BOOTMODE1)"
	P9_23 Alias = "P9_23"
	P9_24 Alias = "P9_24A"
	P9_25 Alias = "P9_25A"
	P9_26 Alias = "P9_26A"
	P9_27 Alias = "P9_27A"
	P9_28 Alias = "P9_28A"
	P9_29 Alias = "P9_29A"
	P9_30 Alias = "P9_30A"
	P9_31 Alias = "P9_31A"
	P9_33 Alias = "P9_33"
	P9_35 Alias = "P9_35"
	P9_36 Alias = "P9_36"
	P9_37 Alias = "P9_37"
	P9_38 Alias = "P9_38"
	P9_39 Alias = "P9_39"
	P9_40 Alias = "P9_40"
	P9_41 Alias = "P9_41"
	P9_42 Alias = "P9_42A"
)

type Number int

type Value int

const (
	LOW  Value = 0
	HIGH Value = 1
)

type Edge string

const (
	NONE    Edge = "none"
	RISING  Edge = "rising"
	FALLING Edge = "falling"
	BOTH    Edge = "both"
)

type Direction string

const (
	IN  Direction = "in"
	OUT Direction = "out"
)

type Pin struct {
	alias     Alias
	number    Number
	direction string
	edge      string
	f         *os.File
}

func (p Pin) String() string {
	return fmt.Sprintf("%d \"%s\"", p.number, p.alias)
}

func (p *Pin) Number() Number {
	return p.number
}

func (p *Pin) Alias() Alias {
	return p.alias
}

func (p *Pin) Poll() (Value, error) {
	if _, err := p.Value(); err != nil {
		return LOW, err
	}
	C.GpioPoll(C.int(p.f.Fd()))
	return p.Value()
}

func (p *Pin) Value() (Value, error) {
	data := make([]byte, 1)
	p.f.Seek(0, 0)
	_, err := p.f.Read(data)
	if err != nil {
		return LOW, err
	}
	value, err := strconv.Atoi(string(data))
	if err != nil {
		return LOW, err
	}
	return Value(value), nil
}

func (p *Pin) SetValue(value Value) error {
	data := fmt.Sprintf("%d", value)
	_, err := p.f.Write([]byte(data))
	return err
}

func (p *Pin) Direction() (Direction, error) {
	data, err := os.ReadFile(p.direction)
	if err != nil {
		return IN, err
	}
	return Direction(data), nil
}

func (p *Pin) SetDirection(direction Direction) error {
	return os.WriteFile(p.direction, []byte(direction), 0666)
}

func (p *Pin) Edge() (Edge, error) {
	data, err := os.ReadFile(p.edge)
	if err != nil {
		return NONE, err
	}
	return Edge(data), nil
}

func (p *Pin) SetEdge(edge Edge) error {
	return os.WriteFile(p.edge, []byte(edge), 0666)
}

func (p *Pin) Unexport() error {
	p.f.Close()
	value := fmt.Sprintf("%d", p.number)
	return os.WriteFile("/sys/class/gpio/unexport", []byte(value), 0666)
}

func Export(alias Alias) (*Pin, error) {
	number, err := GrepNumber(alias)
	if err != nil {
		return nil, err
	}
	value := fmt.Sprintf("%d", number)
	if err := os.WriteFile("/sys/class/gpio/export", []byte(value), 0666); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", number), os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	return &Pin{
		number:    Number(number),
		alias:     alias,
		f:         file,
		direction: fmt.Sprintf("/sys/class/gpio/gpio%d/direction", number),
		edge:      fmt.Sprintf("/sys/class/gpio/gpio%d/edge", number),
	}, nil
}

func GrepNumber(alias Alias) (int, error) {
	cmd := exec.Command(
		"bash", "-c",
		fmt.Sprintf("expr $(ls -l /sys/class/gpio/gpiochip* | grep $(gpiodetect | grep $(gpiofind \"%s\" | grep -o -E \"gpiochip[0-9]+\") | grep -o -E \"[0-9]+\\.gpio\") | grep -o -E \"[0-9]+$\") + $(gpiofind \"%s\" | grep -o -E \"[0-9]+$\")", alias, alias))
	stdout, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	number, err := strconv.Atoi(
		strings.Trim(string(stdout), "\n\r"),
	)
	if err != nil {
		return 0, err
	}
	return number, nil
}
