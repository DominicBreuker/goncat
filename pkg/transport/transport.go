// Package transport defines interfaces for network transport abstractions
// including dialers and listeners for different protocols.
package transport

import "net"

// Dialer is an interface for establishing outbound connections.
type Dialer interface {
	Dial() (net.Conn, error)
}

// Listener is an interface for accepting inbound connections.
type Listener interface {
	Serve(handle Handler) error
	Close() error
}

// Handler is a function that handles an incoming connection.
type Handler func(net.Conn) error
