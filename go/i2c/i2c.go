package i2c

// original: https://gist.github.com/tetsu-koba/33b339d26ac9c730fb09773acf39eac5#file-i2c-go

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
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

func Open(busNumber BusNumber) (c *Bus, err error) {
	path := fmt.Sprintf("/dev/bone/i2c/%d", busNumber)
	f, err := os.OpenFile(path, syscall.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	return &Bus{f: f}, nil
}

func (b *Bus) Close() (err error) {
	return b.f.Close()
}

func (b *Bus) ReadByte(address uint8, offset uint8) (uint8, error) {
	buf := []uint8{0}
	msg := []i2c_msg{
		{
			addr:  uint16(address),
			flags: 0,
			len:   1,
			buf:   uintptr(unsafe.Pointer(&offset)),
		},
		{
			addr:  uint16(address),
			flags: uint16(_I2C_M_RD),
			len:   uint16(len(buf)),
			buf:   uintptr(unsafe.Pointer(&buf[0])),
		},
	}
	err := transfer(b.f, &msg[0], len(msg))
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

func (b *Bus) ReadWord(address uint8, offset uint8) (uint16, error) {
	buf := []uint8{0, 0}
	msg := []i2c_msg{
		{
			addr:  uint16(address),
			flags: 0,
			len:   1,
			buf:   uintptr(unsafe.Pointer(&offset)),
		},
		{
			addr:  uint16(address),
			flags: uint16(_I2C_M_RD),
			len:   uint16(len(buf)),
			buf:   uintptr(unsafe.Pointer(&buf[0])),
		},
	}
	err := transfer(b.f, &msg[0], len(msg))
	if err != nil {
		return 0, err
	}
	word := (uint16(buf[0]) << 8) | uint16(buf[1])
	return word, err
}

func (b *Bus) WriteByte(address uint8, offset uint8, data uint8) error {
	buf := []uint8{offset, data}
	msg := []i2c_msg{
		{
			addr:  uint16(address),
			flags: 0,
			len:   uint16(len(buf)),
			buf:   uintptr(unsafe.Pointer(&buf[0])),
		},
	}
	return transfer(b.f, &msg[0], len(msg))
}

func (b *Bus) WriteWord(address uint8, offset uint8, data uint16) error {
	buf := []uint8{offset, uint8((data >> 8)), uint8(data)}
	msg := []i2c_msg{
		{
			addr:  uint16(address),
			flags: 0,
			len:   uint16(len(buf)),
			buf:   uintptr(unsafe.Pointer(&buf[0])),
		},
	}
	return transfer(b.f, &msg[0], len(msg))
}

const (
	_I2C_RDWR                = 0x0707
	_I2C_RDRW_IOCTL_MAX_MSGS = 42
	_I2C_M_RD                = 0x0001
)

type i2c_msg struct {
	addr      uint16
	flags     uint16
	len       uint16
	__padding uint16
	buf       uintptr
}

type i2c_rdwr_ioctl_data struct {
	msgs  uintptr
	nmsgs uint32
}
