package tcp

import "net"

// MockTCPConn is a mock implementation of net.TCPConn.
type MockTCPConn struct {
	net.Conn
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr
}

// LocalAddr returns the local network address.
func (c *MockTCPConn) LocalAddr() net.Addr {
	if c.localAddr != nil {
		return c.localAddr
	}
	return c.Conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *MockTCPConn) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
	return c.Conn.RemoteAddr()
}

var _ net.Conn = (*MockTCPConn)(nil)
