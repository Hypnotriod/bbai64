//go:build linux
// +build linux

package i2c

import (
	"os"
	"syscall"
	"unsafe"
)

func transfer(f *os.File, msgs *i2c_msg, n int) (err error) {
	data := i2c_rdwr_ioctl_data{
		msgs:  uintptr(unsafe.Pointer(msgs)),
		nmsgs: uint32(n),
	}
	err = nil
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(_I2C_RDWR),
		uintptr(unsafe.Pointer(&data)),
	)
	if errno != 0 {
		err = errno
	}
	return
}
