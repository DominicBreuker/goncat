// Package entrypoint provides entry functions for the four operation modes of goncat.
// These functions encapsulate the logic for starting servers/clients and handlers,
// separating it from CLI argument parsing.
package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	netpkg "dominicbreuker/goncat/pkg/net"
	"dominicbreuker/goncat/pkg/semaphore"
	"errors"
	"fmt"
	"net"
	"sync"
)

// MasterListen starts a server that listens for incoming slave connections
// and controls them as a master.
func MasterListen(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	// Create N=1 semaphore for limiting concurrent stdin/stdout connections.
	// Master listeners are always limited to one connection since they always use stdin/stdout.
	if cfg.Deps == nil {
		cfg.Deps = &config.Dependencies{}
	}
	cfg.Deps.ConnSem = semaphore.New(1, cfg.Timeout)

	return netpkg.ListenAndServe(ctx, cfg, makeMasterHandler(ctx, cfg, mCfg, master.Handle))
}

func makeMasterHandler(
	parent context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	handle masterHandler,
) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		// own context per connection (optional but nice)
		ctx, cancel := context.WithCancel(parent)
		defer cancel()

		var connOnce sync.Once
		closeConn := func() { connOnce.Do(func() { _ = conn.Close() }) }
		defer closeConn()

		// Run the handler directly and return its final error.
		errCh := make(chan error, 1)
		go func() { errCh <- handle(ctx, cfg, mCfg, conn) }()

		select {
		case <-ctx.Done():
			// upstream canceled; close the conn and wait for Handle to finish
			closeConn()
			err := <-errCh
			// treat closure due to our cancel as non-fatal
			if err == nil || errors.Is(err, net.ErrClosed) || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("handling after cancel: %w", err)

		case err := <-errCh:
			// Handler exited on its own
			if err == nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("handling: %w", err)
		}
	}
}
