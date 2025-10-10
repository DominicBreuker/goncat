// Package tcp provides TCP transport implementations.
// It implements the transport.Dialer and transport.Listener interfaces
// for TCP network connections.
package tcp

import (
	"fmt"
	"net"
)

// Dialer implements the transport.Dialer interface for TCP connections.
type Dialer struct {
	tcpAddr *net.TCPAddr
}

// NewDialer creates a new TCP dialer for the specified address.
func NewDialer(addr string) (*Dialer, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	return &Dialer{
		tcpAddr: tcpAddr,
	}, nil
}

// Dial establishes a TCP connection to the configured address with keep-alive enabled.
func (d *Dialer) Dial() (net.Conn, error) {
	conn, err := net.DialTCP("tcp", nil, d.tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.DialTCP(tcp, %s): %s", d.tcpAddr.String(), err)
	}

	conn.SetKeepAlive(true)
	return conn, nil
}
