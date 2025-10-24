//go:build windows

package udp

import (
	"syscall"
)

// setSockoptReuseAddr sets SO_REUSEADDR on the socket.
// Windows version (uses syscall.Handle for file descriptor)
func setSockoptReuseAddr(fd uintptr) error {
	return syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
