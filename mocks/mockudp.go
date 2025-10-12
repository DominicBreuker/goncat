// Package mocks provides mock implementations for testing.
package mocks

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// MockUDPNetwork simulates a UDP network for testing without real network connections.
// It allows creating UDP listeners and dialers that communicate through in-memory channels.
type MockUDPNetwork struct {
	listeners    map[string]*mockUDPListener
	mu           sync.Mutex
	listenerCond *sync.Cond // Condition variable to signal listener changes
}

// NewMockUDPNetwork creates a new mock UDP network.
func NewMockUDPNetwork() *MockUDPNetwork {
	m := &MockUDPNetwork{
		listeners: make(map[string]*mockUDPListener),
	}
	m.listenerCond = sync.NewCond(&m.mu)
	return m
}

// ListenUDP creates a mock UDP listener on the specified address.
func (m *MockUDPNetwork) ListenUDP(network string, laddr *net.UDPAddr) (net.PacketConn, error) {
	if network != "udp" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	addr := laddr.String()
	if _, exists := m.listeners[addr]; exists {
		return nil, fmt.Errorf("address already in use: %s", addr)
	}

	listener := &mockUDPListener{
		addr:    laddr,
		packets: make(chan *mockUDPPacket, 100),
		closeCh: make(chan struct{}),
		network: m,
	}
	m.listeners[addr] = listener
	m.listenerCond.Broadcast() // Signal that a new listener is available

	return listener, nil
}

// ListenPacket creates a mock UDP listener on the specified address.
// This is an alias for ListenUDP to match the net.ListenPacket signature.
func (m *MockUDPNetwork) ListenPacket(network, address string) (net.PacketConn, error) {
	if network != "udp" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	laddr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}

	return m.ListenUDP(network, laddr)
}

// WriteTo sends a UDP packet from one address to another within the mock network.
func (m *MockUDPNetwork) WriteTo(data []byte, srcAddr *net.UDPAddr, dstAddr *net.UDPAddr) (int, error) {
	m.mu.Lock()
	listener, exists := m.listeners[dstAddr.String()]
	m.mu.Unlock()

	if !exists {
		return 0, fmt.Errorf("no listener on %s", dstAddr.String())
	}

	packet := &mockUDPPacket{
		data: make([]byte, len(data)),
		addr: srcAddr,
	}
	copy(packet.data, data)

	select {
	case listener.packets <- packet:
		return len(data), nil
	case <-listener.closeCh:
		return 0, fmt.Errorf("listener closed")
	case <-time.After(1 * time.Second):
		return 0, fmt.Errorf("write timeout")
	}
}

// WaitForListener waits for a listener to be created on the specified address within the given timeout.
// It returns nil if the listener is found, or an error if the timeout expires.
// The timeout is specified in milliseconds.
func (m *MockUDPNetwork) WaitForListener(addr string, timeoutMs int) error {
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
			return fmt.Errorf("timeout waiting for UDP listener on %s", addr)
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

// mockUDPPacket represents a UDP packet in the mock network.
type mockUDPPacket struct {
	data []byte
	addr *net.UDPAddr
}

// mockUDPListener is a mock implementation of net.PacketConn for UDP.
type mockUDPListener struct {
	addr    *net.UDPAddr
	packets chan *mockUDPPacket
	closeCh chan struct{}
	closed  bool
	mu      sync.Mutex
	network *MockUDPNetwork
}

// ReadFrom reads a packet from the connection.
func (l *mockUDPListener) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	select {
	case packet := <-l.packets:
		n = copy(p, packet.data)
		return n, packet.addr, nil
	case <-l.closeCh:
		return 0, nil, fmt.Errorf("connection closed")
	}
}

// WriteTo writes a packet to the specified address.
func (l *mockUDPListener) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return 0, fmt.Errorf("connection closed")
	}
	l.mu.Unlock()

	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return 0, fmt.Errorf("address must be *net.UDPAddr")
	}

	// Find the destination listener and deliver the packet
	l.network.mu.Lock()
	destListener, exists := l.network.listeners[udpAddr.String()]
	l.network.mu.Unlock()

	if !exists {
		// In real UDP, packets can be sent to non-listening addresses
		// For testing purposes, we'll just ignore them
		return len(p), nil
	}

	packet := &mockUDPPacket{
		data: make([]byte, len(p)),
		addr: l.addr,
	}
	copy(packet.data, p)

	select {
	case destListener.packets <- packet:
		return len(p), nil
	case <-destListener.closeCh:
		return len(p), nil // Destination closed, but we sent it
	case <-time.After(100 * time.Millisecond):
		return len(p), nil // Timeout, but pretend we sent it
	}
}

// Close closes the connection.
func (l *mockUDPListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true
	close(l.closeCh)

	// Remove the listener from the network's map
	l.network.mu.Lock()
	delete(l.network.listeners, l.addr.String())
	l.network.mu.Unlock()

	return nil
}

// LocalAddr returns the local network address.
func (l *mockUDPListener) LocalAddr() net.Addr {
	return l.addr
}

// SetDeadline sets the read and write deadlines.
func (l *mockUDPListener) SetDeadline(t time.Time) error {
	// Mock implementation - not used in tests
	return nil
}

// SetReadDeadline sets the read deadline.
func (l *mockUDPListener) SetReadDeadline(t time.Time) error {
	// Mock implementation - not used in tests
	return nil
}

// SetWriteDeadline sets the write deadline.
func (l *mockUDPListener) SetWriteDeadline(t time.Time) error {
	// Mock implementation - not used in tests
	return nil
}

var _ net.PacketConn = (*mockUDPListener)(nil)
