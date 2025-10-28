package net

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
)

// Test helpers and fakes

// fakeConn implements net.Conn for testing.
type fakeConn struct {
	closed  bool
	closeCh chan struct{}
	mu      sync.Mutex
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		closeCh: make(chan struct{}),
	}
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (f *fakeConn) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.closeCh)
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

// Test successful dial for TCP protocol
func TestDial_TCP_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	fakeConn := newFakeConn()
	deps := &dialDependencies{
		dialTCP: func(ctx context.Context, addr string, timeout time.Duration, deps *config.Dependencies) (net.Conn, error) {
			return fakeConn, nil
		},
	}

	conn, err := dial(ctx, cfg, deps)
	if err != nil {
		t.Fatalf("dial() error = %v, want nil", err)
	}
	if conn == nil {
		t.Fatal("dial() returned nil conn")
	}
	if conn != fakeConn {
		t.Error("dial() returned different conn than expected")
	}
}

// Test successful dial for WebSocket protocol
func TestDial_WebSocket_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoWS,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	fakeConn := newFakeConn()
	deps := &dialDependencies{
		dialWS: func(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
			return fakeConn, nil
		},
	}

	conn, err := dial(ctx, cfg, deps)
	if err != nil {
		t.Fatalf("dial() error = %v, want nil", err)
	}
	if conn == nil {
		t.Fatal("dial() returned nil conn")
	}
}

// Test successful dial for WebSocket Secure protocol
func TestDial_WebSocketSecure_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoWSS,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	fakeConn := newFakeConn()
	deps := &dialDependencies{
		dialWSS: func(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
			return fakeConn, nil
		},
	}

	conn, err := dial(ctx, cfg, deps)
	if err != nil {
		t.Fatalf("dial() error = %v, want nil", err)
	}
	if conn == nil {
		t.Fatal("dial() returned nil conn")
	}
}

// Test successful dial for UDP protocol
func TestDial_UDP_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoUDP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	fakeConn := newFakeConn()
	deps := &dialDependencies{
		dialUDP: func(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
			return fakeConn, nil
		},
	}

	conn, err := dial(ctx, cfg, deps)
	if err != nil {
		t.Fatalf("dial() error = %v, want nil", err)
	}
	if conn == nil {
		t.Fatal("dial() returned nil conn")
	}
}

// Test dial failure when connection fails
func TestDial_ConnectionFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	expectedErr := errors.New("connection refused")
	deps := &dialDependencies{
		dialTCP: func(ctx context.Context, addr string, timeout time.Duration, deps *config.Dependencies) (net.Conn, error) {
			return nil, expectedErr
		},
	}

	conn, err := dial(ctx, cfg, deps)
	if err == nil {
		t.Fatal("dial() error = nil, want error")
	}
	if conn != nil {
		t.Error("dial() returned non-nil conn on error")
	}
}

// Test context cancellation
func TestDial_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	deps := &dialDependencies{
		dialTCP: func(ctx context.Context, addr string, timeout time.Duration, deps *config.Dependencies) (net.Conn, error) {
			return nil, context.Canceled
		},
	}

	conn, err := dial(ctx, cfg, deps)
	if err == nil {
		t.Fatal("dial() error = nil, want error")
	}
	if conn != nil {
		t.Error("dial() returned non-nil conn on cancelled context")
	}
}

// Test that public Dial function works
func TestDial_PublicAPI(t *testing.T) {
	t.Skip("Skipping public API test - requires real network")

	// This test would require real network connectivity
	// It's included as a placeholder for manual testing
	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	_, err := Dial(ctx, cfg)
	if err == nil {
		t.Log("Dial succeeded (unexpected unless server is running)")
	} else {
		t.Logf("Dial failed as expected: %v", err)
	}
}
