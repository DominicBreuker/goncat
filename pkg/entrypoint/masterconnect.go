package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dominicbreuker/goncat/pkg/config"
)

// uses interfaces/factories from internal.go (DI for testing)

func MasterConnect(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterConnect(ctx, cfg, mCfg, realClientFactory(), realMasterFactory())
}

func masterConnect(
	parent context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	newClient clientFactory,
	newMaster masterFactory,
) error {
	// Optional: child context to signal normal-return cancellation downstream.
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Build client and connect.
	c := newClient(ctx, cfg)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	var closeOnce sync.Once
	closeClient := func() { closeOnce.Do(func() { _ = c.Close() }) }
	defer closeClient()

	// Create master handler bound to the connected conn.
	h, err := newMaster(ctx, cfg, mCfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("creating master: %w", err)
	}
	defer h.Close()

	// Run Handle and race it against context cancellation.
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.Handle()
	}()

	select {
	case <-ctx.Done():
		// Cancellation: close client/conn and wait for Handle to unwind.
		closeClient()
		err := <-errCh
		// Treat closure due to cancel/close as benign.
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("handling after cancel: %w", err)

	case err := <-errCh:
		// Handle completed on its own.
		if err == nil {
			return nil
		}
		return fmt.Errorf("handling: %w", err)
	}
}
