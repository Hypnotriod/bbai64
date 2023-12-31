//go:build !linux
// +build !linux

package i2c

import (
	"errors"
)

var noImplementationError = errors.New("There is no implementation of i2c bus for this platform!")

func Open(busNumber BusNumber) (c *Bus, err error) {
	return nil, noImplementationError
}

func (b *Bus) Close() (err error) {
	return b.f.Close()
}

func (b *Bus) ReadByte(address uint8, offset uint8) (uint8, error) {
	return 0, noImplementationError
}

func (b *Bus) ReadWord(address uint8, offset uint8) (uint16, error) {
	return 0, noImplementationError
}

func (b *Bus) WriteByte(address uint8, offset uint8, data uint8) error {
	return noImplementationError
}

func (b *Bus) WriteWord(address uint8, offset uint8, data uint16) error {
	return noImplementationError
}
