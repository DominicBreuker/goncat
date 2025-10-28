package net

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
)

// Test successful listen for TCP protocol
func TestListenAndServe_TCP_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	// Mock the transport functions
	deps := &listenDependencies{
		listenAndServeTCP: func(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger, deps *config.Dependencies) error {
			<-ctx.Done()
			return nil
		},
	}

	err := listenAndServe(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("listenAndServe() error = %v, want nil", err)
	}
}

// Test listener failure
func TestListenAndServe_ListenerFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	expectedErr := errors.New("listener failed")
	deps := &listenDependencies{
		listenAndServeTCP: func(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger, deps *config.Dependencies) error {
			return expectedErr
		},
	}

	err := listenAndServe(ctx, cfg, handler, deps)
	if err == nil {
		t.Fatal("listenAndServe() error = nil, want error")
	}
}

// Test WebSocket success
func TestListenAndServe_WebSocket_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := &config.Shared{
		Protocol: config.ProtoWS,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	deps := &listenDependencies{
		listenAndServeWS: func(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
			<-ctx.Done()
			return nil
		},
	}

	err := listenAndServe(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("listenAndServe() error = %v, want nil", err)
	}
}

// Test WebSocket Secure success
func TestListenAndServe_WebSocketSecure_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := &config.Shared{
		Protocol: config.ProtoWSS,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	deps := &listenDependencies{
		listenAndServeWSS: func(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
			<-ctx.Done()
			return nil
		},
	}

	err := listenAndServe(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("listenAndServe() error = %v, want nil", err)
	}
}

// Test UDP success
func TestListenAndServe_UDP_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := &config.Shared{
		Protocol: config.ProtoUDP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	deps := &listenDependencies{
		listenAndServeUDP: func(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
			<-ctx.Done()
			return nil
		},
	}

	err := listenAndServe(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("listenAndServe() error = %v, want nil", err)
	}
}

// Test context cancellation
func TestListenAndServe_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Logger:   log.NewLogger(false),
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	deps := &listenDependencies{
		listenAndServeTCP: func(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger, deps *config.Dependencies) error {
			<-ctx.Done()
			return nil
		},
	}

	// Cancel context after short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := listenAndServe(ctx, cfg, handler, deps)
	// Should return nil on graceful shutdown
	if err != nil {
		t.Fatalf("listenAndServe() error = %v, want nil", err)
	}
}

// Test public API
func TestListenAndServe_PublicAPI(t *testing.T) {
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

	handler := func(conn net.Conn) error {
		return nil
	}

	_ = ListenAndServe(ctx, cfg, handler)
}
