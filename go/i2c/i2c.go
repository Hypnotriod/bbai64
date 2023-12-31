package i2c

// original: https://gist.github.com/tetsu-koba/33b339d26ac9c730fb09773acf39eac5#file-i2c-go

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
