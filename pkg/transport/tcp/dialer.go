// Package tcp provides TCP transport implementations.
// It implements the transport.Dialer and transport.Listener interfaces
// for TCP network connections.
package tcp

import (
	"dominicbreuker/goncat/pkg/config"
	"fmt"
	"net"
)

// Dialer implements the transport.Dialer interface for TCP connections.
type Dialer struct {
	tcpAddr   *net.TCPAddr
	dialerFn  config.TCPDialerFunc
}

// NewDialer creates a new TCP dialer for the specified address.
// The deps parameter is optional and can be nil to use default implementations.
func NewDialer(addr string, deps *config.Dependencies) (*Dialer, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	dialerFn := net.DialTCP
	if deps != nil && deps.TCPDialer != nil {
		dialerFn = deps.TCPDialer
	}

	return &Dialer{
		tcpAddr:  tcpAddr,
		dialerFn: dialerFn,
	}, nil
}

// Dial establishes a TCP connection to the configured address with keep-alive enabled.
func (d *Dialer) Dial() (net.Conn, error) {
	conn, err := d.dialerFn("tcp", nil, d.tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.DialTCP(tcp, %s): %s", d.tcpAddr.String(), err)
	}

	conn.SetKeepAlive(true)
	return conn, nil
}
