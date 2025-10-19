package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"net"
	"testing"
	"time"
)

// uses interfaces from internal.go and fakes from internal_test.go

// it should start the server and handle connections successfully
func TestSlaveListen_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fs := &fakeServer{
		serveCh: make(chan struct{}),
	}
	fsl := &fakeSlave{}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return fsl, nil
	}

	// Run slaveListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveListen(ctx, cfg, newServer, newSlave)
	}()

	// Signal serve to return
	close(fs.serveCh)

	// Wait for completion
	err := <-errCh
	if err != nil {
		t.Fatalf("slaveListen() error = %v, want nil", err)
	}

	if !fs.closed {
		t.Error("server was not closed")
	}
}

// it should return an error if server creation fails
func TestSlaveListen_ServerNewError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	expectedErr := errors.New("server creation failed")
	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return nil, expectedErr
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return nil, nil
	}

	err := slaveListen(ctx, cfg, newServer, newSlave)
	if err == nil {
		t.Fatal("slaveListen() error = nil, want error")
	}
	if !errors.Is(err, expectedErr) && err.Error() != "server.New(): server creation failed" {
		t.Errorf("slaveListen() error = %v, want wrapped server creation error", err)
	}
}

// it should return an error if serving fails
func TestSlaveListen_ServeError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	serveErr := errors.New("serve failed")
	fs := &fakeServer{
		serveErr: serveErr,
	}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return nil, nil
	}

	err := slaveListen(ctx, cfg, newServer, newSlave)
	if err == nil {
		t.Fatal("slaveListen() error = nil, want error")
	}

	if !fs.closed {
		t.Error("server was not closed despite serve error")
	}
}

// it should handle context cancellation by closing the server
func TestSlaveListen_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fs := &fakeServer{
		serveCh: make(chan struct{}),
		closeCh: make(chan struct{}),
	}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return nil, nil
	}

	// Run slaveListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveListen(ctx, cfg, newServer, newSlave)
	}()

	// Cancel context
	cancel()

	// Wait for close to be called (from context cancellation goroutine)
	select {
	case <-fs.closeCh:
		// Good, close was called
	case <-time.After(1 * time.Second):
		t.Error("server Close was not called after context cancellation")
	}

	// Now signal serve to return
	close(fs.serveCh)

	// Wait for completion
	select {
	case <-errCh:
		// Function returned
	case <-time.After(1 * time.Second):
		t.Error("slaveListen did not return after context cancellation")
	}

	if !fs.closed {
		t.Error("server was not closed after context cancellation")
	}
}

// it should ensure Close is idempotent
func TestSlaveListen_CloseIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	fs := &fakeServerWithCloseCount{
		fakeServer: &fakeServer{
			serveCh: make(chan struct{}),
		},
	}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return nil, nil
	}

	// Run slaveListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveListen(ctx, cfg, newServer, newSlave)
	}()

	// Signal serve to return
	close(fs.fakeServer.serveCh)

	// Wait for completion
	<-errCh

	// Close should be called at least once (from defer)
	// May be called twice (defer + context goroutine) but that's OK
	if fs.closeCount < 1 {
		t.Errorf("Close was called %d times, want at least 1", fs.closeCount)
	}
}

// it should create a handler that manages connection lifecycle correctly
func TestMakeSlaveHandler_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
	}

	fsl := &fakeSlave{}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return fsl, nil
	}

	handler := makeSlaveHandler(ctx, cfg, newSlave)
	if handler == nil {
		t.Fatal("makeSlaveHandler returned nil")
	}

	// Create a fake connection
	conn := &fakeConn{}

	err := handler(conn)
	if err != nil {
		t.Error("handler returned error:", err)
	}

	// Connection should be closed even on error
	if !conn.closed {
		t.Error("connection was not closed")
	}
}

// it should close the connection on context cancellation
func TestMakeSlaveHandler_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
	}

	fsl := &fakeSlave{}

	newSlave := func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return fsl, nil
	}

	handler := makeSlaveHandler(ctx, cfg, newSlave)

	conn := &fakeConn{
		closeCh: make(chan struct{}),
	}

	// Start handler in goroutine
	go handler(conn)

	// Cancel context
	cancel()

	// Connection should be closed
	select {
	case <-conn.closeCh:
		// Good, connection was closed
	case <-time.After(1 * time.Second):
		t.Error("connection was not closed after context cancellation")
	}
}
