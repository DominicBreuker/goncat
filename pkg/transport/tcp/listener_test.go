package tcp

import (
	mocks_tcp "dominicbreuker/goncat/mocks/tcp"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/transport"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewListener(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "valid address with port 0",
			addr:    "127.0.0.1:0",
			wantErr: false,
		},
		{
			name:    "wildcard address",
			addr:    ":0",
			wantErr: false,
		},
		{
			name:    "invalid address",
			addr:    "invalid:abc",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Use mock TCP network
			mockNet := mocks_tcp.NewMockTCPNetwork()
			deps := &config.Dependencies{
				TCPListener: mockNet.ListenTCP,
			}

			l, err := NewListener(tc.addr, deps)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewListener(%q) error = %v, wantErr %v", tc.addr, err, tc.wantErr)
			}
			if !tc.wantErr {
				if l == nil {
					t.Error("NewListener() returned nil listener")
				} else {
					l.Close()
				}
			}
		})
	}
}

func TestListener_Serve(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	l, err := NewListener("127.0.0.1:12345", deps)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()
	handlerCalled := make(chan bool, 1)

	handler := func(conn net.Conn) error {
		defer conn.Close()
		handlerCalled <- true
		return nil
	}

	// Start serving in a goroutine
	go func() {
		l.Serve(handler)
	}()

	// Wait for listener to be ready
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Connect to the listener using mock network
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn, err := mockNet.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		t.Fatalf("Failed to connect to listener: %v", err)
	}
	conn.Close()

	// Wait for handler to be called
	select {
	case <-handlerCalled:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Handler was not called")
	}
}

func TestListener_ConcurrentConnections(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	l, err := NewListener("127.0.0.1:12346", deps)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()
	var handlerCount int
	var mu sync.Mutex
	handlerCh := make(chan bool, 10) // Buffered to allow multiple handlers
	handlerStarted := make(chan bool, 10)

	handler := func(conn net.Conn) error {
		defer conn.Close()
		mu.Lock()
		handlerCount++
		mu.Unlock()
		handlerStarted <- true
		<-handlerCh // Block until we signal
		return nil
	}

	// Start serving
	go func() {
		l.Serve(handler)
	}()

	// Wait for listener to be ready
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Connect multiple connections (test with 5 concurrent connections)
	const numConns = 5
	conns := make([]net.Conn, numConns)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)

	for i := 0; i < numConns; i++ {
		conn, err := mockNet.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			t.Fatalf("Failed to connect %d: %v", i, err)
		}
		conns[i] = conn
		defer conn.Close()

		// Wait for handler to start processing
		select {
		case <-handlerStarted:
			// Handler started
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Handler %d did not start", i)
		}
	}

	// All handlers should be running concurrently
	mu.Lock()
	count := handlerCount
	mu.Unlock()
	if count != numConns {
		t.Errorf("Expected %d concurrent handlers, got %d", numConns, count)
	}

	// Signal all handlers to finish
	for i := 0; i < numConns; i++ {
		handlerCh <- true
	}

	// Wait briefly for handlers to complete
	<-time.After(100 * time.Millisecond)
}

func TestListener_HandlerError(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	l, err := NewListener("127.0.0.1:12347", deps)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()
	handlerCalled := make(chan bool, 1)

	handler := func(conn net.Conn) error {
		conn.Close()
		handlerCalled <- true
		return fmt.Errorf("test error")
	}

	go func() {
		l.Serve(handler)
	}()

	// Wait for listener to be ready
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Connect - handler will return error but serve should continue
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn, err := mockNet.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	conn.Close()

	// Wait for first handler to complete
	<-handlerCalled

	// Verify listener is still accepting connections
	conn2, err := mockNet.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		t.Error("Listener stopped accepting after handler error")
	}
	if conn2 != nil {
		conn2.Close()
	}
}

func TestListener_Close(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	l, err := NewListener("127.0.0.1:12348", deps)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}

	addr := l.nl.Addr().String()

	handler := func(conn net.Conn) error {
		conn.Close()
		return nil
	}

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- l.Serve(handler)
	}()

	// Wait for listener to be ready
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Close the listener
	if err := l.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify Serve returns after close
	select {
	case <-serveDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Serve did not return after Close")
	}

	// Verify we can't connect anymore
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn, err := mockNet.DialTCP("tcp", nil, tcpAddr)
	if err == nil {
		// The connection might succeed if there's a race, but it should be closed immediately
		// Try to write to verify it's really closed
		conn.Close()
		t.Error("Expected connection to fail after Close")
	}
}

var _ transport.Listener = (*Listener)(nil)
