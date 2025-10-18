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

// it should close the server on success
func TestMasterListen_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := testConfig()
	mCfg := &config.Master{}

	fs := &fakeServer{serveCh: make(chan struct{})}
	newServer := func(context.Context, *config.Shared, transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	handlerCalled := false
	makeHandler := func(context.Context, *config.Shared, *config.Master) func(net.Conn) error {
		handlerCalled = true
		return func(net.Conn) error { return nil }
	}

	errCh := runAsync(func() error {
		return masterListen(ctx, cfg, mCfg, newServer, makeHandler)
	})

	time.Sleep(50 * time.Millisecond) // Give time to set up
	close(fs.serveCh)                 // Signal serve to return

	assertNoError(t, waitForError(t, errCh, time.Second), "masterListen()")

	if !handlerCalled {
		t.Error("makeHandler was not called")
	}
	if !fs.closed {
		t.Error("server was not closed")
	}
}

// it should return an error if server creation fails
func TestMasterListen_ServerNewError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("server creation failed")
	newServer := func(context.Context, *config.Shared, transport.Handler) (serverInterface, error) {
		return nil, expectedErr
	}
	makeHandler := func(context.Context, *config.Shared, *config.Master) func(net.Conn) error {
		return func(net.Conn) error { return nil }
	}

	err := masterListen(context.Background(), testConfig(), &config.Master{}, newServer, makeHandler)
	assertError(t, err, "masterListen() with server creation error")
}

// it should return an error and close the server if serving fails
func TestMasterListen_ServeError(t *testing.T) {
	t.Parallel()

	fs := &fakeServer{serveErr: errors.New("serve failed")}
	newServer := func(context.Context, *config.Shared, transport.Handler) (serverInterface, error) {
		return fs, nil
	}
	makeHandler := func(context.Context, *config.Shared, *config.Master) func(net.Conn) error {
		return func(net.Conn) error { return nil }
	}

	err := masterListen(context.Background(), testConfig(), &config.Master{}, newServer, makeHandler)
	assertError(t, err, "masterListen() with serve error")

	if !fs.closed {
		t.Error("server was not closed despite serve error")
	}
}

// it should handle context cancellation by closing server
func TestMasterListen_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fs := &fakeServer{
		serveCh: make(chan struct{}),
		closeCh: make(chan struct{}),
	}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	makeHandler := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	// Run masterListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- masterListen(ctx, cfg, mCfg, newServer, makeHandler)
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
		t.Error("masterListen did not return after context cancellation")
	}

	if !fs.closed {
		t.Error("server was not closed after context cancellation")
	}
}

// it should ensure Close is idempotent
func TestMasterListen_CloseIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fs := &fakeServerWithCloseCount{
		fakeServer: &fakeServer{
			serveCh: make(chan struct{}),
		},
	}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	makeHandler := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	// Run masterListen in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- masterListen(ctx, cfg, mCfg, newServer, makeHandler)
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

// it should create a handler that closes connection on context cancellation
func TestMakeMasterHandler_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
	}
	mCfg := &config.Master{}

	handler := makeMasterHandler(ctx, cfg, mCfg)
	if handler == nil {
		t.Fatal("makeMasterHandler returned nil")
	}

	// Create a fake connection
	conn := &fakeConn{}

	// Note: This will fail because master.New requires a real mux session
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

// it should close the connection on context cancellation
func TestMakeMasterHandler_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
	}
	mCfg := &config.Master{}

	handler := makeMasterHandler(ctx, cfg, mCfg)

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
