// Package udp provides UDP transport implementations using QUIC for reliability.
package udp

import (
	"net"
	"time"

	quic "github.com/quic-go/quic-go"
)

// StreamConn adapts a *quic.Stream to the net.Conn interface, allowing QUIC
// streams to be used seamlessly with code that expects net.Conn.
type StreamConn struct {
	conn   *quic.Conn
	stream *quic.Stream
	laddr  net.Addr
	raddr  net.Addr
}

// NewStreamConn creates a new StreamConn wrapping the given QUIC stream.
func NewStreamConn(conn *quic.Conn, stream *quic.Stream, laddr, raddr net.Addr) *StreamConn {
	return &StreamConn{
		conn:   conn,
		stream: stream,
		laddr:  laddr,
		raddr:  raddr,
	}
}

// Read reads data from the stream.
func (c *StreamConn) Read(p []byte) (int, error) {
	return c.stream.Read(p)
}

// Write writes data to the stream.
func (c *StreamConn) Write(p []byte) (int, error) {
	return c.stream.Write(p)
}

// Close closes the stream (closes send side).
func (c *StreamConn) Close() error {
	//return c.stream.Close()
	return c.conn.CloseWithError(0x23, "stream closed")
}

// LocalAddr returns the local network address.
func (c *StreamConn) LocalAddr() net.Addr {
	return c.laddr
}

// RemoteAddr returns the remote network address.
func (c *StreamConn) RemoteAddr() net.Addr {
	return c.raddr
}

// SetDeadline sets both read and write deadlines.
func (c *StreamConn) SetDeadline(t time.Time) error {
	return c.stream.SetDeadline(t)
}

// SetReadDeadline sets the read deadline.
func (c *StreamConn) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline.
func (c *StreamConn) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}
