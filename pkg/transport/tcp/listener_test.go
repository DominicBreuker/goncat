package tcp

import (
	"context"
	mocks_tcp "dominicbreuker/goncat/mocks/tcp"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestListenAndServe_Basic(t *testing.T) {
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

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			handler := func(conn net.Conn) error {
				conn.Close()
				return nil
			}

			errCh := make(chan error, 1)
			go func() {
				errCh <- ListenAndServe(ctx, tc.addr, 10*time.Second, handler, log.NewLogger(false), deps)
			}()

			// Give it a moment to start or fail
			time.Sleep(50 * time.Millisecond)

			// Cancel context to stop server
			cancel()

			// Wait for server to exit
			select {
			case err := <-errCh:
				if (err != nil) != tc.wantErr {
					t.Errorf("ListenAndServe(%q) error = %v, wantErr %v", tc.addr, err, tc.wantErr)
				}
			case <-time.After(1 * time.Second):
				if !tc.wantErr {
					t.Error("ListenAndServe did not exit after context cancellation")
				}
			}
		})
	}
}

func TestListenAndServe_HandlerCalled(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handlerCalled := make(chan bool, 1)
	handler := func(conn net.Conn) error {
		defer conn.Close()
		handlerCalled <- true
		return nil
	}

	// Start server
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- ListenAndServe(ctx, "127.0.0.1:12345", 10*time.Second, handler, log.NewLogger(false), deps)
	}()

	// Wait for listener to be ready
	addr := "127.0.0.1:12345"
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Connect to the listener
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

	// Stop server
	cancel()
	<-serveDone
}

func TestListenAndServe_ConcurrentConnections(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handlerCount int
	var mu sync.Mutex
	handlerCh := make(chan bool, 10)
	handlerStarted := make(chan bool, 10)

	handler := func(conn net.Conn) error {
		defer conn.Close()
		mu.Lock()
		handlerCount++
		mu.Unlock()
		handlerStarted <- true
		<-handlerCh // Block until signaled
		return nil
	}

	// Start server
	go func() {
		ListenAndServe(ctx, "127.0.0.1:12346", 10*time.Second, handler, log.NewLogger(false), deps)
	}()

	// Wait for listener to be ready
	addr := "127.0.0.1:12346"
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Connect multiple connections
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

		// Wait for handler to start
		select {
		case <-handlerStarted:
			// Handler started
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Handler %d did not start", i)
		}
	}

	// Verify all handlers are running
	mu.Lock()
	count := handlerCount
	mu.Unlock()
	if count != numConns {
		t.Errorf("Expected %d concurrent handlers, got %d", numConns, count)
	}

	// Signal handlers to finish
	for i := 0; i < numConns; i++ {
		handlerCh <- true
	}

	// Stop server
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestListenAndServe_HandlerError(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handlerCalled := make(chan bool, 1)
	handler := func(conn net.Conn) error {
		conn.Close()
		handlerCalled <- true
		return fmt.Errorf("test error")
	}

	// Start server
	go func() {
		ListenAndServe(ctx, "127.0.0.1:12347", 10*time.Second, handler, log.NewLogger(false), deps)
	}()

	// Wait for listener to be ready
	addr := "127.0.0.1:12347"
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Connect - handler will return error but server should continue
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn, err := mockNet.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	conn.Close()

	// Wait for handler to complete
	<-handlerCalled

	// Verify listener still accepts connections
	conn2, err := mockNet.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		t.Error("Listener stopped accepting after handler error")
	}
	if conn2 != nil {
		conn2.Close()
	}

	// Stop server
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestListenAndServe_ContextCancellation(t *testing.T) {
	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	ctx, cancel := context.WithCancel(context.Background())

	handler := func(conn net.Conn) error {
		conn.Close()
		return nil
	}

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- ListenAndServe(ctx, "127.0.0.1:12348", 10*time.Second, handler, log.NewLogger(false), deps)
	}()

	// Wait for listener to be ready
	addr := "127.0.0.1:12348"
	if _, err := mockNet.WaitForListener(addr, 1000); err != nil {
		t.Fatalf("Listener not ready: %v", err)
	}

	// Cancel context
	cancel()

	// Verify server exits
	select {
	case err := <-serveDone:
		if err != nil {
			t.Errorf("Expected nil error after cancellation, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("ListenAndServe did not return after context cancellation")
	}

	// Verify we can't connect anymore
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn, err := mockNet.DialTCP("tcp", nil, tcpAddr)
	if err == nil && conn != nil {
		conn.Close()
		t.Error("Expected connection to fail after cancellation")
	}
}
