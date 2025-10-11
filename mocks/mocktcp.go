// Package mocks provides mock implementations for testing.
package mocks

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// MockTCPNetwork simulates a TCP network for testing without real network connections.
// It allows creating listeners and dialers that communicate through in-memory pipes.
type MockTCPNetwork struct {
	listeners    map[string]*mockTCPListener
	mu           sync.Mutex
	listenerCond *sync.Cond // Condition variable to signal listener changes
}

// NewMockTCPNetwork creates a new mock TCP network.
func NewMockTCPNetwork() *MockTCPNetwork {
	m := &MockTCPNetwork{
		listeners: make(map[string]*mockTCPListener),
	}
	m.listenerCond = sync.NewCond(&m.mu)
	return m
}

// ListenTCP creates a mock TCP listener on the specified address.
func (m *MockTCPNetwork) ListenTCP(network string, laddr *net.TCPAddr) (net.Listener, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	addr := laddr.String()
	if _, exists := m.listeners[addr]; exists {
		return nil, fmt.Errorf("address already in use: %s", addr)
	}

	listener := &mockTCPListener{
		addr:    laddr,
		connCh:  make(chan *mockTCPConn, 10),
		closeCh: make(chan struct{}),
		network: m,
	}
	m.listeners[addr] = listener
	m.listenerCond.Broadcast() // Signal that a new listener is available

	return listener, nil
}

// DialTCP creates a mock TCP connection to the specified address.
func (m *MockTCPNetwork) DialTCP(network string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	m.mu.Lock()
	listener, exists := m.listeners[raddr.String()]
	m.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("connection refused: no listener on %s", raddr.String())
	}

	// Create a pair of connected pipes
	clientConn, serverConn := net.Pipe()

	mockClient := &mockTCPConn{
		Conn:       clientConn,
		localAddr:  laddr,
		remoteAddr: raddr,
	}
	mockServer := &mockTCPConn{
		Conn:       serverConn,
		localAddr:  raddr,
		remoteAddr: laddr,
	}

	// Send the server side to the listener
	select {
	case listener.connCh <- mockServer:
		// Connection established
	case <-listener.closeCh:
		clientConn.Close()
		serverConn.Close()
		return nil, fmt.Errorf("connection refused: listener closed")
	case <-time.After(1 * time.Second):
		clientConn.Close()
		serverConn.Close()
		return nil, fmt.Errorf("connection timeout")
	}

	return mockClient, nil
}

// WaitForListener waits for a listener to be created on the specified address within the given timeout.
// It returns nil if the listener is found, or an error if the timeout expires.
// The timeout is specified in milliseconds.
func (m *MockTCPNetwork) WaitForListener(addr string, timeoutMs int) error {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		// Check if the listener already exists
		if _, exists := m.listeners[addr]; exists {
			return nil
		}

		// Check if we've exceeded the timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for listener on %s", addr)
		}

		// Wait for a signal that a new listener is available, with a small timeout
		// to periodically check the deadline
		go func() {
			time.Sleep(50 * time.Millisecond)
			m.listenerCond.Broadcast()
		}()
		m.listenerCond.Wait()
	}
}

// mockTCPListener is a mock implementation of net.TCPListener.
type mockTCPListener struct {
	addr    *net.TCPAddr
	connCh  chan *mockTCPConn
	closeCh chan struct{}
	closed  bool
	mu      sync.Mutex
	network *MockTCPNetwork
}

// Accept waits for and returns the next connection to the listener.
func (l *mockTCPListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connCh:
		return conn, nil
	case <-l.closeCh:
		return nil, fmt.Errorf("listener closed")
	}
}

// Close closes the listener.
func (l *mockTCPListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true
	close(l.closeCh)
	return nil
}

// Addr returns the listener's network address.
func (l *mockTCPListener) Addr() net.Addr {
	return l.addr
}

// mockTCPConn is a mock implementation of net.TCPConn.
type mockTCPConn struct {
	net.Conn
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr
}

// LocalAddr returns the local network address.
func (c *mockTCPConn) LocalAddr() net.Addr {
	if c.localAddr != nil {
		return c.localAddr
	}
	return c.Conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *mockTCPConn) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
	return c.Conn.RemoteAddr()
}

var _ net.Listener = (*mockTCPListener)(nil)
var _ net.Conn = (*mockTCPConn)(nil)
