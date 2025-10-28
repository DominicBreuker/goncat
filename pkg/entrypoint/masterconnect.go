package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	netpkg "dominicbreuker/goncat/pkg/net"
)

// uses interfaces/factories from internal.go (DI for testing)

func MasterConnect(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterConnect(ctx, cfg, mCfg, netpkg.Dial, master.Handle)
}

func masterConnect(
	parent context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	dial func(context.Context, *config.Shared) (net.Conn, error),
	handle masterHandler,
) error {
	// Optional: child context to signal normal-return cancellation downstream.
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

	// Run the master handler
	errCh := make(chan error, 1)
	go func() {
		errCh <- handle(ctx, cfg, mCfg, conn)
	}()

	select {
	case <-ctx.Done():
		// Cancellation: close conn and wait for Handle to unwind.
		cfg.Logger.VerboseMsg("Master connect: context cancelled, closing connection")
		closeConn()
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
