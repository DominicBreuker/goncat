package slave

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestNewUDPRelay_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	// Create a pipe for the remote connection
	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	if relay == nil {
		t.Fatal("NewUDPRelay() returned nil")
	}

	if relay.ConnLocal == nil {
		t.Error("Local UDP connection not created")
	}

	if relay.ConnRemote != serverConn {
		t.Error("Remote connection not set correctly")
	}
}

func TestNewUDPRelay_GetChannelError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	expectedErr := errors.New("channel error")
	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return nil, expectedErr
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl)
	if err == nil {
		if relay != nil {
			relay.Close()
		}
		t.Error("Expected error when GetOneChannel fails, got nil")
	}
}

func TestUDPRelay_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}

	// First close should succeed
	err = relay.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify isClosed flag is set
	if !relay.isClosed {
		t.Error("isClosed flag not set after Close()")
	}

	// Second close should also work
	_ = relay.Close()
}

func TestUDPRelay_LogError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	// LogError should not panic
	relay.LogError("test error message")
	relay.LogError("test with args: %s %d", "string", 123)
}

func TestUDPRelay_SendToDst_WhenClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}

	// Close the relay first
	relay.Close()

	// Attempting to send should fail
	err = relay.sendToDst("127.0.0.1", 8080, []byte("test"))
	if err == nil {
		t.Error("Expected error when sending to closed relay, got nil")
	}
}

func TestUDPRelay_SendToDst_InvalidAddr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	// Use invalid address format
	err = relay.sendToDst("invalid address", -1, []byte("test"))
	if err == nil {
		t.Error("Expected error with invalid address, got nil")
	}
}

func TestUDPRelay_ContextPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	if relay.ctx == nil {
		t.Error("Context not set in relay")
	}

	// Cancel context
	cancel()

	// Context in relay should be cancelled
	select {
	case <-relay.ctx.Done():
		// Success - context was cancelled
	default:
		t.Error("Relay context not cancelled when parent cancelled")
	}
}
