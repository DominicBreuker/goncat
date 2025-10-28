package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	netpkg "dominicbreuker/goncat/pkg/net"
	"dominicbreuker/goncat/pkg/semaphore"
	"errors"
	"fmt"
	"net"
	"sync"
)

func SlaveListen(ctx context.Context, cfg *config.Shared) error {
	// Create N=1 semaphore for limiting concurrent stdin/stdout connections.
	// Slave listeners accept multiple command execution sessions but only one stdin/stdout piping session.
	if cfg.Deps == nil {
		cfg.Deps = &config.Dependencies{}
	}
	cfg.Deps.ConnSem = semaphore.New(1, cfg.Timeout)

	return netpkg.ListenAndServe(ctx, cfg, makeSlaveHandler(ctx, cfg, slave.Handle))
}

func makeSlaveHandler(
	parent context.Context,
	cfg *config.Shared,
	handle slaveHandler,
) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		// per-connection context
		ctx, cancel := context.WithCancel(parent)
		defer cancel()

		var connOnce sync.Once
		closeConn := func() { connOnce.Do(func() { _ = conn.Close() }) }
		defer closeConn()

		// run the handler directly; it will manage the connection lifecycle
		errCh := make(chan error, 1)
		go func() { errCh <- handle(ctx, cfg, conn) }()

		select {
		case <-ctx.Done():
			closeConn()
			err := <-errCh
			if err == nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("handling after cancel: %w", err)

		case err := <-errCh:
			if err == nil {
				return nil
			}
			return fmt.Errorf("handling: %w", err)
		}
	}
}
