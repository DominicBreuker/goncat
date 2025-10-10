package ws

import (
	"context"
	"dominicbreuker/goncat/pkg/transport"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestNewListener(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		tls     bool
		wantErr bool
	}{
		{
			name:    "valid address without TLS",
			addr:    "127.0.0.1:0",
			tls:     false,
			wantErr: false,
		},
		{
			name:    "valid address with TLS",
			addr:    "127.0.0.1:0",
			tls:     true,
			wantErr: false,
		},
		{
			name:    "wildcard address",
			addr:    ":0",
			tls:     false,
			wantErr: false,
		},
		{
			name:    "invalid address",
			addr:    "invalid:abc",
			tls:     false,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			l, err := NewListener(ctx, tc.addr, tc.tls)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewListener() error = %v, wantErr %v", err, tc.wantErr)
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

	ctx := context.Background()
	l, err := NewListener(ctx, "127.0.0.1:0", false)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()
	handlerCalled := make(chan bool, 1)
	serverReady := make(chan bool)

	handler := func(conn net.Conn) error {
		defer conn.Close()
		handlerCalled <- true
		return nil
	}

	// Start serving in a goroutine
	go func() {
		serverReady <- true
		l.Serve(handler)
	}()

	// Wait for server to be ready
	<-serverReady

	// Connect to the listener
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws://" + addr
	c, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	})
	if err != nil {
		t.Fatalf("Failed to connect to listener: %v", err)
	}
	c.Close(websocket.StatusNormalClosure, "")

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

	ctx := context.Background()
	l, err := NewListener(ctx, "127.0.0.1:0", false)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()
	handlerCh := make(chan bool)
	handlerStarted := make(chan bool)
	serverReady := make(chan bool)

	handler := func(conn net.Conn) error {
		defer conn.Close()
		handlerStarted <- true
		<-handlerCh // Block until we signal
		return nil
	}

	// Start serving
	go func() {
		serverReady <- true
		l.Serve(handler)
	}()

	// Wait for server to be ready
	<-serverReady

	// Connect first connection
	dialCtx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	wsURL := "ws://" + addr
	c1, resp1, err := websocket.Dial(dialCtx1, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	})
	if err != nil {
		t.Fatalf("Failed to connect first time: %v", err)
	}
	defer c1.Close(websocket.StatusNormalClosure, "")
	if resp1.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("First connection status = %d, want %d", resp1.StatusCode, http.StatusSwitchingProtocols)
	}

	// Wait for handler to start processing
	<-handlerStarted

	// Try second connection - should be rejected with 500
	dialCtx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	c2, resp2, err := websocket.Dial(dialCtx2, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	})
	if err == nil {
		c2.Close(websocket.StatusNormalClosure, "")
	}
	// Second connection should receive HTTP 500
	if resp2.StatusCode == http.StatusSwitchingProtocols {
		t.Error("Second connection should not have been upgraded")
	}

	// Signal first handler to finish
	handlerCh <- true
}

func TestListener_HandlerError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	ctx := context.Background()
	l, err := NewListener(ctx, "127.0.0.1:0", false)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}
	defer l.Close()

	addr := l.nl.Addr().String()
	handlerCalled := make(chan bool, 1)
	serverReady := make(chan bool)

	handler := func(conn net.Conn) error {
		conn.Close()
		handlerCalled <- true
		return nil
	}

	go func() {
		serverReady <- true
		l.Serve(handler)
	}()

	// Wait for server to be ready
	<-serverReady

	// Connect - handler will close connection
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws://" + addr
	c, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	c.Close(websocket.StatusNormalClosure, "")

	// Wait for first handler to complete
	<-handlerCalled

	// Verify listener is still accepting connections
	c2, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	})
	if err != nil {
		t.Error("Listener stopped accepting after handler error")
	}
	if c2 != nil {
		c2.Close(websocket.StatusNormalClosure, "")
	}
}

func TestListener_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	ctx := context.Background()
	l, err := NewListener(ctx, "127.0.0.1:0", false)
	if err != nil {
		t.Fatalf("NewListener() error = %v", err)
	}

	addr := l.nl.Addr().String()

	handler := func(conn net.Conn) error {
		conn.Close()
		return nil
	}

	serveDone := make(chan error, 1)
	serverReady := make(chan bool)
	go func() {
		serverReady <- true
		serveDone <- l.Serve(handler)
	}()

	// Wait for server to be ready
	<-serverReady

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
	dialCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	wsURL := "ws://" + addr
	c, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	})
	if err == nil {
		c.Close(websocket.StatusNormalClosure, "")
		t.Error("Expected connection to fail after Close")
	}
}

var _ transport.Listener = (*Listener)(nil)
