package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
)

// uses interfaces/factories from internal.go (DI for testing)

func MasterConnect(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterConnect(ctx, cfg, mCfg, realClientFactory(), master.Handle)
}

func masterConnect(
	parent context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	newClient clientFactory,
	handle masterHandler,
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

	// Run the master handler (newMaster now runs the handler directly and returns its final error).
	errCh := make(chan error, 1)
	go func() {
		errCh <- handle(ctx, cfg, mCfg, c.GetConnection())
	}()

	select {
	case <-ctx.Done():
		// Cancellation: close client/conn and wait for Handle to unwind.
		cfg.Logger.VerboseMsg("Master connect: context cancelled, closing connection")
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
