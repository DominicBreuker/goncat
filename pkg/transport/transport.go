// Package transport provides abstractions for network transport protocols.
// It defines common interfaces for establishing connections (Dialer) and
// accepting incoming connections (Listener) that can be implemented by
// different transport protocols such as TCP and WebSocket.
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
