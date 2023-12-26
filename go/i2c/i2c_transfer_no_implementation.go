//go:build !linux
// +build !linux

package i2c

import (
	"errors"
	"os"
)

func transfer(f *os.File, msgs *i2c_msg, n int) (err error) {
	return errors.New("There is no implementation of i2c bus for this platform!")
}
