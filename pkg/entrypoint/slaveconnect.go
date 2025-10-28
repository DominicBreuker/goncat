package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	netpkg "dominicbreuker/goncat/pkg/net"
)

// uses interfaces/factories from internal.go (DI for testing)

func SlaveConnect(ctx context.Context, cfg *config.Shared) error {
	return slaveConnect(ctx, cfg, netpkg.Dial, slave.Handle)
}

func slaveConnect(
	parent context.Context,
	cfg *config.Shared,
	dial func(context.Context, *config.Shared) (net.Conn, error),
	handle slaveHandler,
) error {
	// Optional: child context so we can cancel descendants on normal return
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	// Dial connection directly
	conn, err := dial(ctx, cfg)
	if err != nil {
		return fmt.Errorf("dialing: %w", err)
	}
	var closeOnce sync.Once
	closeConn := func() { closeOnce.Do(func() { _ = conn.Close() }) }
	defer closeConn()

	// Run the slave handler directly
	errCh := make(chan error, 1)
	go func() { errCh <- handle(ctx, cfg, conn) }()

	select {
	case <-ctx.Done():
		// Cancellation: close and wait for Handle to exit
		cfg.Logger.VerboseMsg("Slave connect: context cancelled, closing connection")
		closeConn()
		err := <-errCh
		// Treat cancel/close as benign
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("handling after cancel: %w", err)

	case err := <-errCh:
		// Handle completed on its own
		if err == nil {
			return nil
		}
		return fmt.Errorf("handling: %w", err)
	}
}
