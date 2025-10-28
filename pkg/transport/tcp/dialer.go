// Package tcp provides TCP transport implementations.
// It provides stateless functions for establishing TCP connections (Dial)
// and serving incoming connections (ListenAndServe).
package tcp

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"fmt"
	"net"
	"time"
)

// Dial establishes a TCP connection to the specified address.
// The connection has keep-alive and no-delay enabled for optimal performance.
// Accepts a context for cancellation, address, timeout, and optional dependencies.
// Returns the established connection or an error.
//
// The timeout parameter is used to set a deadline before the dial operation.
// The deadline is cleared immediately after the operation completes to prevent
// the timeout from affecting subsequent operations on the connection.
func Dial(ctx context.Context, addr string, timeout time.Duration, deps *config.Dependencies) (net.Conn, error) {
	// Parse address
	tcpAddr, err := resolveTCPAddress(addr)
	if err != nil {
		return nil, err
	}

	// Get dialer function from dependencies
	dialerFn := config.GetTCPDialerFunc(deps)

	// Establish connection with timeout
	conn, err := dialWithTimeout(ctx, dialerFn, tcpAddr, timeout)
	if err != nil {
		return nil, err
	}

	// Configure connection for optimal performance
	configureConnection(conn)

	return conn, nil
}

// resolveTCPAddress parses and resolves a TCP address string.
func resolveTCPAddress(addr string) (*net.TCPAddr, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}
	return tcpAddr, nil
}

// dialWithTimeout establishes a TCP connection with proper timeout handling.
// Sets deadline before dial, clears it immediately after to prevent affecting
// subsequent operations on the connection.
func dialWithTimeout(ctx context.Context, dialerFn config.TCPDialerFunc, tcpAddr *net.TCPAddr, timeout time.Duration) (net.Conn, error) {
	conn, err := dialerFn(ctx, "tcp", nil, tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.DialTCP(tcp, %s): %w", tcpAddr.String(), err)
	}

	// Clear any deadlines that may have been set during dial
	// This is critical - lingering deadlines can kill healthy connections
	if timeout > 0 {
		_ = conn.SetDeadline(time.Time{})
	}

	return conn, nil
}

// configureConnection sets optimal TCP options on the connection.
func configureConnection(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetNoDelay(true)
	}
}
