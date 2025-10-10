// Package format provides utility functions for formatting network addresses and other data.
package format

import (
	"fmt"
	"strings"
)

// Addr formats a host and port into a network address string.
// IPv6 addresses are enclosed in brackets, e.g., "[::1]:8080".
// IPv4 addresses are formatted as "host:port", e.g., "127.0.0.1:8080".
func Addr(host string, port int) string {
	if strings.ContainsAny(host, ":") { // IPv6
		return fmt.Sprintf("[%s]:%d", host, port)
	} else { // IPv4
		return fmt.Sprintf("%s:%d", host, port)
	}
}
