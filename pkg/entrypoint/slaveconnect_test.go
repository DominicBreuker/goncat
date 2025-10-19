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
	fs := &fakeSlave{}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return fs, nil
	}

	err := slaveConnect(ctx, cfg, newClient, newSlave)
	if err != nil {
		t.Fatalf("slaveConnect() error = %v, want nil", err)
	}

	if !fc.closed {
		t.Error("client was not closed")
	}

	if !fs.closed {
		t.Error("slave handler was not closed")
	}
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

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		t.Error("newSlave should not be called when connect fails")
		return nil, nil
	}

	err := slaveConnect(ctx, cfg, newClient, newSlave)
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

// it should return an error and close the client if slave creation fails
func TestSlaveConnect_SlaveNewError(t *testing.T) {
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
	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return nil, slaveNewErr
	}

	err := slaveConnect(ctx, cfg, newClient, newSlave)
	if err == nil {
		t.Fatal("slaveConnect() error = nil, want error")
	}

	// Client should still be closed even when slave creation fails
	if !fc.closed {
		t.Error("client was not closed despite slave.New error")
	}
}

// it should return an error and close both client and slave if handling fails
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

	handleErr := errors.New("handle failed")
	fs := &fakeSlave{
		handleErr: handleErr,
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return fs, nil
	}

	err := slaveConnect(ctx, cfg, newClient, newSlave)
	if err == nil {
		t.Fatal("slaveConnect() error = nil, want error")
	}

	// Both client and slave should be closed even on handle error
	if !fc.closed {
		t.Error("client was not closed despite handle error")
	}
	if !fs.closed {
		t.Error("slave was not closed despite handle error")
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
	fs := &fakeSlave{
		handleFunc: func() error {
			<-handleCh // Block until we signal
			return nil
		},
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return fs, nil
	}

	// Run slaveConnect in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveConnect(ctx, cfg, newClient, newSlave)
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
