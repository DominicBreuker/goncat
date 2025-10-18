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

// fakeServer implements a fake server for testing.
type fakeServer struct {
	serveErr  error
	serveCh   chan struct{}
	closed    bool
	closeCh   chan struct{}
	closeErr  error
	closeWait time.Duration
}

func (f *fakeServer) Serve() error {
	if f.serveCh != nil {
		<-f.serveCh
	}
	return f.serveErr
}

func (f *fakeServer) Close() error {
	if f.closeWait > 0 {
		time.Sleep(f.closeWait)
	}
	if !f.closed {
		f.closed = true
		if f.closeCh != nil {
			close(f.closeCh)
		}
	}
	return f.closeErr
}

func TestMasterListen_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fs := &fakeServer{
		serveCh: make(chan struct{}),
	}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	handlerCalled := false
	makeHandler := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(net.Conn) error {
		handlerCalled = true
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
	close(fs.serveCh)

	// Wait for completion
	err := <-errCh
	if err != nil {
		t.Fatalf("masterListen() error = %v, want nil", err)
	}

	if !handlerCalled {
		t.Error("makeHandler was not called")
	}

	if !fs.closed {
		t.Error("server was not closed")
	}
}

func TestMasterListen_ServerNewError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	expectedErr := errors.New("server creation failed")
	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return nil, expectedErr
	}

	makeHandler := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	err := masterListen(ctx, cfg, mCfg, newServer, makeHandler)
	if err == nil {
		t.Fatal("masterListen() error = nil, want error")
	}
	if !errors.Is(err, expectedErr) && err.Error() != "server.New(): server creation failed" {
		t.Errorf("masterListen() error = %v, want wrapped server creation error", err)
	}
}

func TestMasterListen_ServeError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	serveErr := errors.New("serve failed")
	fs := &fakeServer{
		serveErr: serveErr,
	}

	newServer := func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return fs, nil
	}

	makeHandler := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(net.Conn) error {
		return func(conn net.Conn) error {
			return nil
		}
	}

	err := masterListen(ctx, cfg, mCfg, newServer, makeHandler)
	if err == nil {
		t.Fatal("masterListen() error = nil, want error")
	}

	if !fs.closed {
		t.Error("server was not closed despite serve error")
	}
}

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

// fakeConn implements a fake net.Conn for testing.
type fakeConn struct {
	closed  bool
	closeCh chan struct{}
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *fakeConn) Close() error {
	if !f.closed {
		f.closed = true
		if f.closeCh != nil {
			close(f.closeCh)
		}
	}
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (f *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}
}

func (f *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (f *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (f *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// fakeServerWithCloseCount wraps fakeServer to count Close calls.
type fakeServerWithCloseCount struct {
	*fakeServer
	closeCount int
}

func (f *fakeServerWithCloseCount) Close() error {
	f.closeCount++
	return f.fakeServer.Close()
}
