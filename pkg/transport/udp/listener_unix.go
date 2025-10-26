//go:build unix

package udp

import (
	"syscall"
)

// setSockoptReuseAddr sets SO_REUSEADDR on the socket.
// Unix version (Linux, macOS, BSD, etc.)
func setSockoptReuseAddr(fd uintptr) error {
	return syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
