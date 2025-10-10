package tcp

import (
	"dominicbreuker/goncat/pkg/transport"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestNewListener(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

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
			l, err := NewListener(tc.addr)
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
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	l, err := NewListener("127.0.0.1:0")
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

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the listener
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect to listener: %v", err)
	}
	conn.Close()

	// Wait for handler to be called
	select {
	case <-handlerCalled:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Handler was not called")
	}
}

func TestListener_SingleConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	l, err := NewListener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()
	handlerCount := 0
	handlerCh := make(chan bool)

	handler := func(conn net.Conn) error {
		defer conn.Close()
		handlerCount++
		<-handlerCh // Block until we signal
		return nil
	}

	// Start serving
	go func() {
		l.Serve(handler)
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect first connection
	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn1.Close()

	// Give handler time to be called
	time.Sleep(100 * time.Millisecond)

	// Try second connection - should be rejected since first is still active
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Second connection should be closed immediately
	time.Sleep(100 * time.Millisecond)
	buf := make([]byte, 1)
	conn2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _ = conn2.Read(buf) // Intentionally ignoring error - we're checking if connection closes
	conn2.Close()

	// Signal first handler to finish
	handlerCh <- true

	// Verify only one handler was called
	time.Sleep(100 * time.Millisecond)
	if handlerCount != 1 {
		t.Errorf("Expected 1 handler call, got %d", handlerCount)
	}
}

func TestListener_HandlerError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	l, err := NewListener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()

	handler := func(conn net.Conn) error {
		conn.Close()
		return fmt.Errorf("test error")
	}

	go func() {
		l.Serve(handler)
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect - handler will return error but serve should continue
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	conn.Close()

	// Give time for error handling
	time.Sleep(100 * time.Millisecond)

	// Verify listener is still accepting connections
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Error("Listener stopped accepting after handler error")
	}
	if conn2 != nil {
		conn2.Close()
	}
}

func TestListener_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	l, err := NewListener("127.0.0.1:0")
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

	time.Sleep(100 * time.Millisecond)

	// Close the listener
	if err := l.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify Serve returns after close
	select {
	case <-serveDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Serve did not return after Close")
	}

	// Verify we can't connect anymore
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		conn.Close()
		t.Error("Expected connection to fail after Close")
	}
}

var _ transport.Listener = (*Listener)(nil)
