package slave

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"net"
	"strings"
)

// ClientControlSession ...
type ClientControlSession interface {
	GetOneChannel() (net.Conn, error)
	Send(m msg.Message) error
}

// isErrorHostUnreachable is my personal list of Go errors that shall count as the SOCKS5 "Host unreachable" error
func isErrorHostUnreachable(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "no such host")
}

// isErrorConnectionRefused is my personal list of Go errors that shall count as the SOCKS5 "Connection Refused" error
func isErrorConnectionRefused(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "connection refused") ||
		strings.HasSuffix(s, "host is down")
}

// isErrorNetworkUnreachable is my personal list of Go errors that shall count as the SOCKS5 "Network unreachable" error
func isErrorNetworkUnreachable(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "network is unreachable")
}
