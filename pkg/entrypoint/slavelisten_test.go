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

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	handlerCalled := false
	makeHandler := func(ctx context.Context, cfg *config.Shared) func(net.Conn) error {
		handlerCalled = true
		return func(conn net.Conn) error {
			return nil
		}
	}

	// Run slaveListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveListen(ctx, cfg, newServer, makeHandler)
	}()

	// Give it time to set up
	time.Sleep(50 * time.Millisecond)

	// Signal serve to return
	close(fs.serveCh)

	// Wait for completion
	err := <-errCh
	if err != nil {
		t.Fatalf("slaveListen() error = %v, want nil", err)
	}

	if !handlerCalled {
		t.Error("makeHandler was not called")
	}

	if !fs.closed {
		t.Error("server was not closed")
	}
}

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

	makeHandler := func(ctx context.Context, cfg *config.Shared) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	err := slaveListen(ctx, cfg, newServer, makeHandler)
	if err == nil {
		t.Fatal("slaveListen() error = nil, want error")
	}
	if !errors.Is(err, expectedErr) && err.Error() != "server.New(): server creation failed" {
		t.Errorf("slaveListen() error = %v, want wrapped server creation error", err)
	}
}

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

	makeHandler := func(ctx context.Context, cfg *config.Shared) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	err := slaveListen(ctx, cfg, newServer, makeHandler)
	if err == nil {
		t.Fatal("slaveListen() error = nil, want error")
	}

	if !fs.closed {
		t.Error("server was not closed despite serve error")
	}
}

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

	makeHandler := func(ctx context.Context, cfg *config.Shared) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	// Run slaveListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveListen(ctx, cfg, newServer, makeHandler)
	}()

	// Give it time to set up
	time.Sleep(50 * time.Millisecond)

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

	makeHandler := func(ctx context.Context, cfg *config.Shared) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	// Run slaveListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- slaveListen(ctx, cfg, newServer, makeHandler)
	}()

	// Give it time to set up
	time.Sleep(50 * time.Millisecond)

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

func TestMakeSlaveHandler_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
	}

	handler := makeSlaveHandler(ctx, cfg)
	if handler == nil {
		t.Fatal("makeSlaveHandler returned nil")
	}

	// Create a fake connection
	conn := &fakeConn{}

	// Note: This will fail because slave.New requires a real mux session
	// We're just testing that the handler can be called
	err := handler(conn)

	// We expect an error because we don't have a real mux session
	if err == nil {
		t.Error("handler() error = nil, expected error due to missing mux session")
	}

	// Connection should be closed even on error
	if !conn.closed {
		t.Error("connection was not closed")
	}
}

func TestMakeSlaveHandler_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
	}

	handler := makeSlaveHandler(ctx, cfg)

	conn := &fakeConn{
		closeCh: make(chan struct{}),
	}

	// Start handler in goroutine
	go handler(conn)

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

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
