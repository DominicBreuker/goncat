//go:build linux
// +build linux

package pty

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func ptsname(f *os.File) (string, error) {
	n, err := unix.IoctlGetInt(int(f.Fd()), unix.TIOCGPTN)
	return fmt.Sprintf("/dev/pts/%d", n), err
}

func grantpt(f *os.File) error {
	var n uintptr
	return ioctl(f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
}

func unlockpt(f *os.File) error {
	var u uintptr
	return ioctl(f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&u)))
}
