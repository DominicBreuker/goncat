package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
)

// uses interfaces/factories from internal.go (DI for testing)

func SlaveConnect(ctx context.Context, cfg *config.Shared) error {
	return slaveConnect(ctx, cfg, realClientFactory(), slave.Handle)
}

func slaveConnect(
	parent context.Context,
	cfg *config.Shared,
	newClient clientFactory,
	handle slaveHandler,
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

	// Run the slave handler directly (it accepts the conn and runs until finished).
	errCh := make(chan error, 1)
	go func() { errCh <- handle(ctx, cfg, c.GetConnection()) }()

	select {
	case <-ctx.Done():
		// Cancellation: close and wait for Handle to exit
		cfg.Logger.VerboseMsg("Slave connect: context cancelled, closing connection")
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
