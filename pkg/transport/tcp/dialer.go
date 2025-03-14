package tcp

import (
	"fmt"
	"net"
)

// Dialer ...
type Dialer struct {
	tcpAddr *net.TCPAddr
}

// NewDialer ...
func NewDialer(addr string) (*Dialer, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	return &Dialer{
		tcpAddr: tcpAddr,
	}, nil
}

// Dial ...
func (d *Dialer) Dial() (net.Conn, error) {
	conn, err := net.DialTCP("tcp", nil, d.tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.DialTCP(tcp, %s): %s", d.tcpAddr.String(), err)
	}

	conn.SetKeepAlive(true)
	return conn, nil
}
