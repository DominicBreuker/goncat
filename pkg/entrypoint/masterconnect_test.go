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

// it should close the connection and master handler on success
func TestMasterConnect_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fakeConn := &fakeConn{}

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return fakeConn, nil
	}

	// Inline fake master handler: returns nil (success)
	err := masterConnect(ctx, cfg, mCfg, dial, newFakeMasterHandle(nil, nil))
	if err != nil {
		t.Fatalf("masterConnect() error = %v, want nil", err)
	}

	if !fakeConn.closed {
		t.Error("connection was not closed")
	}
}

// it should return an error if dialing fails
func TestMasterConnect_DialError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	dialErr := errors.New("connection failed")

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return nil, dialErr
	}

	err := masterConnect(ctx, cfg, mCfg, dial, newFakeMasterHandle(nil, func() {
		t.Error("handler should not be called when dial fails")
	}))
	if err == nil {
		t.Fatal("masterConnect() error = nil, want error")
	}
	if !errors.Is(err, dialErr) && err.Error() != "dialing: connection failed" {
		t.Errorf("masterConnect() error = %v, want wrapped dial error", err)
	}
}

// it should return an error if handling fails
func TestMasterConnect_HandleError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fakeConn := &fakeConn{}

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return fakeConn, nil
	}

	handleErr := errors.New("handle failed")

	err := masterConnect(ctx, cfg, mCfg, dial, newFakeMasterHandle(handleErr, nil))
	if err == nil {
		t.Fatal("masterConnect() error = nil, want error")
	}

	// Connection should be closed even on handle error
	if !fakeConn.closed {
		t.Error("connection was not closed despite handle error")
	}
}

// it should handle context cancellation by closing connection and master handler
func TestMasterConnect_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fakeConn := &fakeConn{
		closeCh: make(chan struct{}),
	}

	dial := func(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
		return fakeConn, nil
	}

	// Master that blocks in Handle
	handleCh := make(chan struct{})

	// Run masterConnect in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- masterConnect(ctx, cfg, mCfg, dial, newFakeMasterHandle(nil, func() {
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
		t.Error("masterConnect did not return")
	}
}
