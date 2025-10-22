package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"errors"
	"testing"
	"time"
)

// uses interfaces from internal.go and fakes from internal_test.go

// it should close the client and slave handler on success
func TestSlaveConnect_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fc := &fakeClient{
		conn: &fakeConn{},
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	err := slaveConnect(ctx, cfg, newClient, newFakeSlaveHandle(nil, nil))
	if err != nil {
		t.Fatalf("slaveConnect() error = %v, want nil", err)
	}

	if !fc.closed {
		t.Error("client was not closed")
	}

	// Note: the entrypoint does not manage handler.Close(); the handler itself
	// is responsible for cleanup. We only assert client cleanup here.
}

// it should return an error if connecting fails
func TestSlaveConnect_ConnectError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	connectErr := errors.New("connection failed")
	fc := &fakeClient{
		connectErr: connectErr,
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	err := slaveConnect(ctx, cfg, newClient, newFakeSlaveHandle(nil, func() {
		t.Error("handler should not be called when connect fails")
	}))
	if err == nil {
		t.Fatal("slaveConnect() error = nil, want error")
	}
	if !errors.Is(err, connectErr) && err.Error() != "connecting: connection failed" {
		t.Errorf("slaveConnect() error = %v, want wrapped connection error", err)
	}

	if fc.closed {
		t.Error("client should not be closed when connect fails")
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

	fc := &fakeClient{
		conn: &fakeConn{},
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	slaveNewErr := errors.New("slave creation failed")

	err := slaveConnect(ctx, cfg, newClient, newFakeSlaveHandle(slaveNewErr, nil))
	if err == nil {
		t.Fatal("slaveConnect() error = nil, want error")
	}

	// Client should still be closed even when slave creation fails
	if !fc.closed {
		t.Error("client was not closed despite slave.New error")
	}
}

// it should handle context cancellation by closing client and slave
func TestSlaveConnect_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fc := &fakeClient{
		conn:    &fakeConn{},
		closeCh: make(chan struct{}),
	}

	// Slave that blocks in Handle
	handleCh := make(chan struct{})

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	// Run slaveConnect in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveConnect(ctx, cfg, newClient, newFakeSlaveHandle(nil, func() {
			<-handleCh
		}))
	}()

	// Cancel context
	cancel()

	// Client should be closed
	select {
	case <-fc.closeCh:
		// Good, client was closed
	case <-time.After(1 * time.Second):
		t.Error("client Close was not called after context cancellation")
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
