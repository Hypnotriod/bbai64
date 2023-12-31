package i2c

import (
	"os"
)

type BusNumber int

const (
	Bus1 BusNumber = 1
	Bus2 BusNumber = 2
	Bus3 BusNumber = 3
	Bus4 BusNumber = 4
)

type Bus struct {
	f *os.File
}

const DevicePath = "/dev/bone/i2c/%d"
