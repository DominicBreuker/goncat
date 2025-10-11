package config

import (
	"io"
	"net"
	"os"
)

// Dependencies contains injectable dependencies for testing and customization.
// All fields are optional and will use default implementations if nil.
type Dependencies struct {
	TCPDialer   TCPDialerFunc
	TCPListener TCPListenerFunc
	Stdin       StdinFunc
	Stdout      StdoutFunc
}

// TCPDialerFunc is a function that dials a TCP connection.
// It returns a net.Conn to allow for mock implementations.
type TCPDialerFunc func(network string, laddr, raddr *net.TCPAddr) (net.Conn, error)

// TCPListenerFunc is a function that creates a TCP listener.
// It returns a net.Listener to allow for mock implementations.
type TCPListenerFunc func(network string, laddr *net.TCPAddr) (net.Listener, error)

// StdinFunc is a function that returns a reader for stdin.
// It returns an io.Reader to allow for mock implementations.
type StdinFunc func() io.Reader

// StdoutFunc is a function that returns a writer for stdout.
// It returns an io.Writer to allow for mock implementations.
type StdoutFunc func() io.Writer

// GetTCPDialerFunc returns the TCP dialer function from dependencies, or a default implementation.
// If deps is nil or deps.TCPDialer is nil, returns a function that uses net.DialTCP.
func GetTCPDialerFunc(deps *Dependencies) TCPDialerFunc {
	if deps != nil && deps.TCPDialer != nil {
		return deps.TCPDialer
	}
	return func(network string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
		return net.DialTCP(network, laddr, raddr)
	}
}

// GetTCPListenerFunc returns the TCP listener function from dependencies, or a default implementation.
// If deps is nil or deps.TCPListener is nil, returns a function that uses net.ListenTCP.
func GetTCPListenerFunc(deps *Dependencies) TCPListenerFunc {
	if deps != nil && deps.TCPListener != nil {
		return deps.TCPListener
	}
	return func(network string, laddr *net.TCPAddr) (net.Listener, error) {
		return net.ListenTCP(network, laddr)
	}
}

// GetStdinFunc returns the stdin function from dependencies, or a default implementation.
// If deps is nil or deps.Stdin is nil, returns a function that uses os.Stdin.
func GetStdinFunc(deps *Dependencies) StdinFunc {
	if deps != nil && deps.Stdin != nil {
		return deps.Stdin
	}
	return func() io.Reader {
		return os.Stdin
	}
}

// GetStdoutFunc returns the stdout function from dependencies, or a default implementation.
// If deps is nil or deps.Stdout is nil, returns a function that uses os.Stdout.
func GetStdoutFunc(deps *Dependencies) StdoutFunc {
	if deps != nil && deps.Stdout != nil {
		return deps.Stdout
	}
	return func() io.Writer {
		return os.Stdout
	}
}
