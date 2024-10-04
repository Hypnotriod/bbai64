package gpio

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Alias string

const (
	P8_02 Alias = "P8_02"
	P8_03 Alias = "P8_03"
	P8_04 Alias = "P8_04"
	P8_05 Alias = "P8_05"
	P8_06 Alias = "P8_06"
	P8_07 Alias = "P8_07"
	P8_08 Alias = "P8_08"
	P8_09 Alias = "P8_09"
	P8_10 Alias = "P8_10"
	P8_11 Alias = "P8_11"
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
	P8_31 Alias = "P8_31"
	P8_32 Alias = "P8_32"
	P8_33 Alias = "P8_33"
	P8_34 Alias = "P8_34"
	P8_35 Alias = "P8_35"
	P8_36 Alias = "P8_36"
	P8_37 Alias = "P8_37"
	P8_38 Alias = "P8_38"
	P8_39 Alias = "P8_39"
	P8_40 Alias = "P8_40"
	P8_41 Alias = "P8_41"
	P8_42 Alias = "P8_42"
	P8_43 Alias = "P8_43"
	P8_44 Alias = "P8_44"
	P8_45 Alias = "P8_45"
	P8_46 Alias = "P8_46"
	P9_11 Alias = "P9_11"
	P9_12 Alias = "P9_12"
	P9_13 Alias = "P9_13"
	P9_14 Alias = "P9_14"
	P9_15 Alias = "P9_15"
	P9_16 Alias = "P9_16"
	P9_17 Alias = "P9_17"
	P9_18 Alias = "P9_18"
	P9_19 Alias = "P9_19"
	P9_20 Alias = "P9_20"
	P9_21 Alias = "P9_21"
	P9_22 Alias = "P9_22"
	P9_23 Alias = "P9_23"
	P9_24 Alias = "P9_24"
	P9_25 Alias = "P9_25"
	P9_26 Alias = "P9_26"
	P9_27 Alias = "P9_27"
	P9_28 Alias = "P9_28"
	P9_29 Alias = "P9_29"
	P9_30 Alias = "P9_30"
	P9_31 Alias = "P9_31"
	P9_33 Alias = "P9_33"
	P9_35 Alias = "P9_35"
	P9_36 Alias = "P9_36"
	P9_37 Alias = "P9_37"
	P9_38 Alias = "P9_38"
	P9_39 Alias = "P9_39"
	P9_40 Alias = "P9_40"
	P9_41 Alias = "P9_41"
	P9_42 Alias = "P9_42"
)

type Number int

type Value int

const (
	LOW  Value = 0
	HIGH Value = 1
)

type Direction string

const (
	IN  Direction = "in"
	OUT Direction = "out"
)

type Gpio struct {
	alias     Alias
	number    Number
	direction string
	value     string
}

func (g *Gpio) Number() Number {
	return g.number
}

func (g *Gpio) Alias() Alias {
	return g.alias
}

func (g *Gpio) Value() (Value, error) {
	data, err := os.ReadFile(g.value)
	if err != nil {
		return LOW, err
	}
	value, err := strconv.Atoi(string(data))
	if err != nil {
		return LOW, err
	}
	return Value(value), nil
}

func (g *Gpio) SetValue(value Value) error {
	data := fmt.Sprintf("%d", value)
	return os.WriteFile(g.value, []byte(data), 0666)
}

func (g *Gpio) Direction() (Direction, error) {
	data, err := os.ReadFile(g.direction)
	if err != nil {
		return IN, err
	}
	return Direction(data), nil
}

func (g *Gpio) SetDirection(direction Direction) error {
	return os.WriteFile(g.direction, []byte(direction), 0666)
}

func (g *Gpio) Unexport() error {
	value := fmt.Sprintf("%d", g.number)
	return os.WriteFile("/sys/class/gpio/unexport", []byte(value), 0666)
}

func Export(alias Alias) (*Gpio, error) {
	number, err := GrepNumber(alias)
	if err != nil {
		return nil, err
	}
	value := fmt.Sprintf("%d", number)
	if err := os.WriteFile("/sys/class/gpio/export", []byte(value), 0666); err != nil {
		return nil, err
	}
	return &Gpio{
		number:    Number(number),
		alias:     alias,
		value:     fmt.Sprintf("/sys/class/gpio/gpio%d/value", number),
		direction: fmt.Sprintf("/sys/class/gpio/gpio%d/direction", number),
	}, nil
}

func GrepNumber(alias Alias) (int, error) {
	cmd := exec.Command(
		"bash", "-c",
		fmt.Sprintf("expr $(ls -l /sys/class/gpio/gpiochip* | grep $(gpiodetect | grep $(gpiofind %s | grep -o -E \"gpiochip[0-9]+\") | grep -o -E \"[0-9]+\\.gpio\") | grep -o -E \"[0-9]+$\") + $(gpiofind %s | grep -o -E \"[0-9]+$\")", alias, alias))
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
