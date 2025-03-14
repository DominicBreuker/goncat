package transport

import "net"

type Dialer interface {
	Dial() (net.Conn, error)
}

type Listener interface {
	Serve(handle Handler) error
	Close() error
}

type Handler func(net.Conn) error
