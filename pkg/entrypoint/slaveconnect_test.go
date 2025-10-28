package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"errors"
	"net"
	"testing"
	"time"
)

// uses interfaces from internal.go and fakes from internal_test.go

// it should close the connection and slave handler on success
func TestSlaveConnect_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fakeConn := &fakeConn{}

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return fakeConn, nil
	}

	err := slaveConnect(ctx, cfg, dial, newFakeSlaveHandle(nil, nil))
	if err != nil {
		t.Fatalf("slaveConnect() error = %v, want nil", err)
	}

	if !fakeConn.closed {
		t.Error("connection was not closed")
	}
}

// it should return an error if dialing fails
func TestSlaveConnect_DialError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	dialErr := errors.New("connection failed")

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return nil, dialErr
	}

	err := slaveConnect(ctx, cfg, dial, newFakeSlaveHandle(nil, func() {
		t.Error("handler should not be called when dial fails")
	}))
	if err == nil {
		t.Fatal("slaveConnect() error = nil, want error")
	}
	if !errors.Is(err, dialErr) && err.Error() != "dialing: connection failed" {
		t.Errorf("slaveConnect() error = %v, want wrapped dial error", err)
	}
}

// it should return an error if slave handling fails
func TestSlaveConnect_HandleError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fakeConn := &fakeConn{}

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return fakeConn, nil
	}

	slaveNewErr := errors.New("slave creation failed")

	err := slaveConnect(ctx, cfg, dial, newFakeSlaveHandle(slaveNewErr, nil))
	if err == nil {
		t.Fatal("slaveConnect() error = nil, want error")
	}

	// Connection should still be closed even when slave creation fails
	if !fakeConn.closed {
		t.Error("connection was not closed despite slave.New error")
	}
}

// it should handle context cancellation by closing connection and slave
func TestSlaveConnect_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fakeConn := &fakeConn{
		closeCh: make(chan struct{}),
	}

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return fakeConn, nil
	}

	// Slave that blocks in Handle
	handleCh := make(chan struct{})

	// Run slaveConnect in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveConnect(ctx, cfg, dial, newFakeSlaveHandle(nil, func() {
			<-handleCh
		}))
	}()

	// Cancel context
	cancel()

	// Connection should be closed
	select {
	case <-fakeConn.closeCh:
		// Good, connection was closed
	case <-time.After(1 * time.Second):
		t.Error("connection Close was not called after context cancellation")
	}

	// Now signal handle to return
	close(handleCh)

	// Wait for completion
	select {
	case <-errCh:
		// Function returned
	case <-time.After(1 * time.Second):
		t.Error("slaveConnect did not return")
	}
}
