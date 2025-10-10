//go:build linux || darwin
// +build linux darwin

package pty

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// NewPty creates a new pseudo-terminal pair on Unix systems.
// It returns the master (ptm) and slave (pts) file descriptors.
// The caller is responsible for closing both file descriptors.
func NewPty() (*os.File, *os.File, error) {
	ptm, err := openPtm()
	if err != nil {
		return nil, nil, fmt.Errorf("openMaster(): %s", err)
	}

	ptsName, err := ptsname(ptm)
	if err != nil {
		ptm.Close()
		return nil, nil, fmt.Errorf("getPtsName(ptm): %s", err)
	}

	if err := grantpt(ptm); err != nil {
		ptm.Close()
		return nil, nil, fmt.Errorf("grantpt(ptm): %s", err)
	}

	if err := unlockpt(ptm); err != nil {
		ptm.Close()
		return nil, nil, fmt.Errorf("unlockpt(ptm): %s", err)
	}

	pts, err := openPts(ptsName)
	if err != nil {
		ptm.Close()
		return nil, nil, fmt.Errorf("openPts(%s): %s", ptsName, err)
	}
	return ptm, pts, nil
}

func openPts(ptsName string) (*os.File, error) {
	pts, err := os.OpenFile(ptsName, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile: %s", err)
	}

	return pts, nil
}

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if e != 0 {
		return e
	}
	return nil
}

// Terminal size

type winsize struct {
	rows uint16
	cols uint16
	x    uint16
	y    uint16
}

// GetTerminalSize retrieves the current terminal size from stdout.
// It returns the terminal dimensions or an error if the ioctl call fails.
func GetTerminalSize() (size TerminalSize, err error) {
	var ws winsize
	if err := ioctl(os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws))); err != nil {
		return size, err
	}

	return TerminalSize{
		Rows: int(ws.rows),
		Cols: int(ws.cols),
	}, nil
}

// SetTerminalSize sets the terminal size for the given PTY file descriptor.
// It uses the TIOCSWINSZ ioctl to update the terminal dimensions.
func SetTerminalSize(t *os.File, size TerminalSize) error {
	ws := &winsize{
		rows: uint16(size.Rows),
		cols: uint16(size.Cols),
	}
	return ioctl(t.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(ws)))
}
