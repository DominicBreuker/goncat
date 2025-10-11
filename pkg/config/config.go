// Package config defines configuration structures and validation logic
// for the goncat application, including protocol definitions, shared settings,
// and master/slave specific configurations.
package config

import (
	"fmt"
	"io"
	"net"
	"os"
)

// Shared contains configuration settings common to both master and slave modes,
// including network protocol, connection details, and security settings.
type Shared struct {
	Protocol Protocol
	Host     string
	Port     int
	SSL      bool
	Key      string
	Verbose  bool
	Deps     *Dependencies
}

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

// Protocol represents the network protocol type used for communication.
type Protocol int

// Protocol type constants.
const (
	ProtoTCP = 1 // ProtoTCP represents plain TCP protocol
	ProtoWS  = 2 // ProtoWS represents WebSocket protocol without TLS
	ProtoWSS = 3 // ProtoWSS represents WebSocket protocol with TLS
)

// String returns the string representation of the Protocol.
func (p Protocol) String() string {
	switch p {
	case ProtoTCP:
		return "tcp"
	case ProtoWS:
		return "ws"
	case ProtoWSS:
		return "wss"
	default:
		return ""
	}
}

// KeySalt is a random salt value used in key derivation.
// This value is overwritten with a random value during release builds via ldflags.
var KeySalt = "98263df478dbb76e25eed7e71750e59dbffcb1f401413472f9b128f10bb3cc01af3942a17980a24cd1a26bd3ab87a0fec835faf59aa4f1a1dc7f2416c5765e9e"

// Validate checks the Shared configuration for errors.
// It returns a slice of validation errors, or an empty slice if valid.
func (c *Shared) Validate() []error {
	var errors []error

	if !c.SSL && c.Key != "" {
		errors = append(errors, fmt.Errorf("you must use '--ssl' to use '--key'"))
	}

	if err := validatePort(c.Port); err != nil {
		errors = append(errors, fmt.Errorf("'--port': %s", err))
	}

	return errors
}

// GetKey returns the salted key for authentication.
// If no key is configured, it returns an empty string.
// Otherwise, it returns the KeySalt concatenated with the configured key.
func (c *Shared) GetKey() string {
	if c.Key == "" {
		return ""
	}

	return KeySalt + c.Key
}

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
