package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dominicbreuker/goncat/pkg/config"
)

// uses interfaces/factories from internal.go (DI for testing)

func SlaveConnect(ctx context.Context, cfg *config.Shared) error {
	return slaveConnect(ctx, cfg, realClientFactory(), realSlaveFactory())
}

func slaveConnect(
	parent context.Context,
	cfg *config.Shared,
	newClient clientFactory,
	newSlave slaveFactory,
) error {
	// Optional: child context so we can cancel descendants on normal return
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Build client and connect
	c := newClient(ctx, cfg)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	var closeOnce sync.Once
	closeClient := func() { closeOnce.Do(func() { _ = c.Close() }) }
	defer closeClient()

	// Create slave handler bound to the connected conn
	h, err := newSlave(ctx, cfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("creating slave: %w", err)
	}
	defer h.Close()

	// Run Handle and race against context cancellation
	errCh := make(chan error, 1)
	go func() { errCh <- h.Handle() }()

	select {
	case <-ctx.Done():
		// Cancellation: close and wait for Handle to exit
		closeClient()
		err := <-errCh
		// Treat cancel/close as benign
		if err == nil || errors.Is(err, context.Canceled) /* || errors.Is(err, net.ErrClosed) */ {
			return nil
		}
		return fmt.Errorf("handling after cancel: %w", err)

	case err := <-errCh:
		// Handle completed on its own
		if err == nil /* || errors.Is(err, net.ErrClosed) */ {
			return nil
		}
		return fmt.Errorf("handling: %w", err)
	}
}
