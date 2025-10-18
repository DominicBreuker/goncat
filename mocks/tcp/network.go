// Package tcp provides mock TCP network primitives for testing.
package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// MockTCPNetwork simulates a TCP network for testing without real network connections.
// It allows creating listeners and dialers that communicate through in-memory pipes.
type MockTCPNetwork struct {
	listeners    map[string]*MockTCPListener
	mu           sync.Mutex
	listenerCond *sync.Cond // Condition variable to signal listener changes
}

// NewMockTCPNetwork creates a new mock TCP network.
func NewMockTCPNetwork() *MockTCPNetwork {
	m := &MockTCPNetwork{
		listeners: make(map[string]*MockTCPListener),
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

	listener := &MockTCPListener{
		addr:       laddr,
		connCh:     make(chan *MockTCPConn, 10),
		acceptedCh: make(chan *MockTCPConn, 16),
		closeCh:    make(chan struct{}),
		network:    m,
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

	// If laddr is nil, generate a mock ephemeral local address
	if laddr == nil {
		laddr = &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 50000 + (int(time.Now().UnixNano()) % 10000), // Mock ephemeral port
		}
	}

	// Create a pair of connected pipes
	clientConn, serverConn := net.Pipe()

	mockClient := &MockTCPConn{
		Conn:       clientConn,
		localAddr:  laddr,
		remoteAddr: raddr,
	}
	mockServer := &MockTCPConn{
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

// DialTCPContext is a context-aware wrapper compatible with the new TCPDialerFunc signature.
// It simply ignores the context (mock network is in-memory) but preserves the behavior and timeout semantics.
func (m *MockTCPNetwork) DialTCPContext(ctx context.Context, network string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
	// If context is already done, return quickly
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return m.DialTCP(network, laddr, raddr)
}

// WaitForListener waits for a listener to be created on the specified address within the given timeout.
// It returns nil if the listener is found, or an error if the timeout expires.
// The timeout is specified in milliseconds.
func (m *MockTCPNetwork) WaitForListener(addr string, timeoutMs int) (*MockTCPListener, error) {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		// Check if the listener already exists
		if l, exists := m.listeners[addr]; exists {
			return l, nil
		}

		// Check if we've exceeded the timeout
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for listener on %s", addr)
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
