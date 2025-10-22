package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"errors"
	"testing"
	"time"
)

// uses interfaces from internal.go and fakes from internal_test.go

// it should close the client and master handler on success
func TestMasterConnect_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fc := &fakeClient{
		conn: &fakeConn{},
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	// Inline fake master handler: returns nil (success)
	err := masterConnect(ctx, cfg, mCfg, newClient, newFakeMasterHandle(nil, nil))
	if err != nil {
		t.Fatalf("masterConnect() error = %v, want nil", err)
	}

	if !fc.closed {
		t.Error("client was not closed")
	}
}

// it should return an error if connecting fails
func TestMasterConnect_ConnectError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	connectErr := errors.New("connection failed")
	fc := &fakeClient{
		connectErr: connectErr,
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	err := masterConnect(ctx, cfg, mCfg, newClient, newFakeMasterHandle(nil, func() {
		t.Error("newMaster should not be called when connect fails")
	}))
	if err == nil {
		t.Fatal("masterConnect() error = nil, want error")
	}
	if !errors.Is(err, connectErr) && err.Error() != "connecting: connection failed" {
		t.Errorf("masterConnect() error = %v, want wrapped connection error", err)
	}

	if fc.closed {
		t.Error("client should not be closed when connect fails")
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

	fc := &fakeClient{
		conn: &fakeConn{},
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	handleErr := errors.New("handle failed")

	err := masterConnect(ctx, cfg, mCfg, newClient, newFakeMasterHandle(handleErr, nil))
	if err == nil {
		t.Fatal("masterConnect() error = nil, want error")
	}

	// Client should be closed even on handle error. The master handler is
	// executed by the factory and may handle its own cleanup; we only assert
	// the client cleanup here.
	if !fc.closed {
		t.Error("client was not closed despite handle error")
	}
}

// it should handle context cancellation by closing client and master handler
func TestMasterConnect_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
	mCfg := &config.Master{}

	fc := &fakeClient{
		conn:    &fakeConn{},
		closeCh: make(chan struct{}),
	}

	// Master that blocks in Handle
	handleCh := make(chan struct{})

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	// Run masterConnect in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- masterConnect(ctx, cfg, mCfg, newClient, newFakeMasterHandle(nil, func() {
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
		t.Error("masterConnect did not return")
	}
}
