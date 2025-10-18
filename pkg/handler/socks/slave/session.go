// Package slave provides the slave-side implementation for handling SOCKS5 proxy
// requests. It receives SOCKS5 connection and association requests from the master
// and establishes the actual connections to target destinations.
package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"net"
	"strings"
)

// ClientControlSession represents the interface for communicating over
// a multiplexed control session to handle SOCKS5 proxy requests from the master.
type ClientControlSession interface {
	GetOneChannelContext(ctx context.Context) (net.Conn, error)
	// SendContext sends a control message and respects the provided context for
	// cancellation. Implementations should return promptly when ctx is done.
	SendContext(ctx context.Context, m msg.Message) error
}

// isErrorHostUnreachable checks if the error indicates a "host unreachable" condition
// that should be mapped to the SOCKS5 "Host unreachable" error code.
func isErrorHostUnreachable(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "no such host")
}

// isErrorConnectionRefused checks if the error indicates a "connection refused" condition
// that should be mapped to the SOCKS5 "Connection refused" error code.
func isErrorConnectionRefused(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "connection refused") ||
		strings.HasSuffix(s, "host is down")
}

// isErrorNetworkUnreachable checks if the error indicates a "network unreachable" condition
// that should be mapped to the SOCKS5 "Network unreachable" error code.
func isErrorNetworkUnreachable(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "network is unreachable")
}
