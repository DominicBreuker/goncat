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
	fm := &fakeMaster{}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	newMaster := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (handlerInterface, error) {
		return fm, nil
	}

	err := masterConnect(ctx, cfg, mCfg, newClient, newMaster)
	if err != nil {
		t.Fatalf("masterConnect() error = %v, want nil", err)
	}

	if !fc.closed {
		t.Error("client was not closed")
	}

	if !fm.closed {
		t.Error("master handler was not closed")
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

	newMaster := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (handlerInterface, error) {
		t.Error("newMaster should not be called when connect fails")
		return nil, nil
	}

	err := masterConnect(ctx, cfg, mCfg, newClient, newMaster)
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

// it should return an error if master creation fails
func TestMasterConnect_MasterNewError(t *testing.T) {
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

	masterNewErr := errors.New("master creation failed")
	newMaster := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (handlerInterface, error) {
		return nil, masterNewErr
	}

	err := masterConnect(ctx, cfg, mCfg, newClient, newMaster)
	if err == nil {
		t.Fatal("masterConnect() error = nil, want error")
	}

	// Client should still be closed even when master creation fails
	if !fc.closed {
		t.Error("client was not closed despite master.New error")
	}
}

// it should return an error and close the master handler and client if handling fails
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

	handleErr := errors.New("handle failed")
	fm := &fakeMaster{
		handleErr: handleErr,
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	newMaster := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (handlerInterface, error) {
		return fm, nil
	}

	err := masterConnect(ctx, cfg, mCfg, newClient, newMaster)
	if err == nil {
		t.Fatal("masterConnect() error = nil, want error")
	}

	// Both client and master should be closed even on handle error
	if !fc.closed {
		t.Error("client was not closed despite handle error")
	}
	if !fm.closed {
		t.Error("master was not closed despite handle error")
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
	fm := &fakeMaster{
		handleFunc: func() error {
			<-handleCh // Block until we signal
			return nil
		},
	}

	newClient := func(ctx context.Context, cfg *config.Shared) clientInterface {
		return fc
	}

	newMaster := func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (handlerInterface, error) {
		return fm, nil
	}

	// Run masterConnect in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- masterConnect(ctx, cfg, mCfg, newClient, newMaster)
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
