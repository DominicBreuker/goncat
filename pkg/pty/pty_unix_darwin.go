//go:build darwin
// +build darwin

package pty

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// compare:
// https://github.com/xyproto/vt100/blob/main/vendor/github.com/pkg/term/termios/pty_darwin.go
// https://opensource.apple.com/source/xnu/xnu-792.2.4/bsd/sys/ioccom.h.auto.html
// https://opensource.apple.com/source/Libc/Libc-825.26/stdlib/grantpt.c.auto.html

func openPtm() (*os.File, error) {
	ptmFd, err := syscall.Open("/dev/ptmx", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf(" syscall.Open(/dev/ptmx): %s", err)
	}
	return os.NewFile(uintptr(ptmFd), "/dev/ptmx"), nil
}

const _IOCPARM_MASK = 0x1fff
const _IOCPARM_LEN = (syscall.TIOCPTYGNAME >> 16) & _IOCPARM_MASK

func ptsname(f *os.File) (string, error) {
	buf := make([]byte, _IOCPARM_LEN)

	err := ioctl(f.Fd(), syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&buf[0])))
	if err != nil {
		return "", fmt.Errorf("ioctl(fd, TIOCPTYGNAME, buf): %s", err)
	}

	name, err := bytesToString(buf)
	if err != nil {
		return "", fmt.Errorf("bytesToString(buf): %s", err)
	}

	return name, nil
}

func bytesToString(buf []byte) (string, error) {
	n := bytes.IndexByte(buf, 0)
	if n == -1 {
		return "", fmt.Errorf("no null byte in buffer")
	}

	return string(buf[:n]), nil
}

func grantpt(f *os.File) error {
	return ioctl(f.Fd(), syscall.TIOCPTYGRANT, 0)
}

func unlockpt(f *os.File) error {
	return ioctl(f.Fd(), syscall.TIOCPTYUNLK, 0)
}
