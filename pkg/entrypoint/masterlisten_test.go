package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"errors"
	"testing"
	"time"
)

// Test that MasterListen initializes semaphore correctly
func TestMasterListen_InitializesSemaphore(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := testConfig()
	mCfg := &config.Master{}

	// This will fail quickly because no server is actually listening,
	// but we can verify the semaphore was initialized
	if cfg.Deps == nil {
		cfg.Deps = &config.Dependencies{}
	}

	// Run in background since it will block
	errCh := make(chan error, 1)
	go func() {
		errCh <- MasterListen(ctx, cfg, mCfg)
	}()

	// Wait for context timeout or completion
	select {
	case <-errCh:
	// Function returned
	case <-time.After(100 * time.Millisecond):
		// Timeout - that's okay
	}

	// Verify semaphore was created
	if cfg.Deps.ConnSem == nil {
		t.Error("ConnSem was not initialized")
	}
}

// Test that makeMasterHandler creates a working handler
func TestMakeMasterHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := testConfig()
	mCfg := &config.Master{}

	fakeConn := &fakeConn{}

	handler := makeMasterHandler(ctx, cfg, mCfg, newFakeMasterHandle(nil, nil))

	err := handler(fakeConn)
	if err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}

	if !fakeConn.closed {
		t.Error("connection was not closed")
	}
}

// Test handler with error
func TestMakeMasterHandler_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := testConfig()
	mCfg := &config.Master{}

	fakeConn := &fakeConn{}

	handler := makeMasterHandler(ctx, cfg, mCfg, newFakeMasterHandle(errors.New("handle error"), nil))

	err := handler(fakeConn)
	if err == nil {
		t.Fatal("handler() error = nil, want error")
	}

	if !fakeConn.closed {
		t.Error("connection was not closed despite error")
	}
}

// Test handler with context cancellation
func TestMakeMasterHandler_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := testConfig()
	mCfg := &config.Master{}

	fakeConn := &fakeConn{
		closeCh: make(chan struct{}),
	}

	handleCh := make(chan struct{})
	handler := makeMasterHandler(ctx, cfg, mCfg, newFakeMasterHandle(nil, func() {
		<-handleCh
	}))

	// Run handler in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- handler(fakeConn)
	}()

	// Cancel context
	cancel()

	// Wait for connection to be closed
	select {
	case <-fakeConn.closeCh:
	// Good
	case <-time.After(time.Second):
		t.Error("connection was not closed after context cancellation")
	}

	// Signal handler to complete
	close(handleCh)

	// Wait for handler to complete
	select {
	case <-errCh:
	// Completed
	case <-time.After(time.Second):
		t.Error("handler did not return")
	}
}
