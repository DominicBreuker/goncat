// Package entrypoint provides entry functions for the four operation modes of goncat.
// These functions encapsulate the logic for starting servers/clients and handlers,
// separating it from CLI argument parsing.
package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"errors"
	"fmt"
	"net"
	"sync"
)

// uses interfaces/factories from internal.go (DI for testing)

// MasterListen starts a server that listens for incoming slave connections
// and controls them as a master.
func MasterListen(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterListen(ctx, cfg, mCfg, realServerFactory(), master.Handle)
}

// masterListen is the internal implementation that accepts injected dependencies for testing.
func masterListen(
	parent context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	newServer serverFactory,
	handle masterHandler,
) error {
	// child ctx we will cancel on return
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	s, err := newServer(ctx, cfg, makeMasterHandler(ctx, cfg, mCfg, handle))
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	var closeOnce sync.Once
	closeServer := func() { closeOnce.Do(func() { _ = s.Close() }) }
	defer closeServer()

	// run Serve in a goroutine
	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve() }()

	select {
	case <-ctx.Done():
		// our context got canceled (parent canceled or defer cancel on return)
		closeServer()
		// wait for Serve to exit
		err := <-errCh
		// if Close() caused Serve() to return ErrClosed (or your sentinel), treat as graceful
		if err == nil || isServerClosed(err) || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("serving after cancel: %w", err)

	case err := <-errCh:
		// Serve exited on its own
		if err == nil || isServerClosed(err) {
			return nil
		}
		// propagate real error
		return fmt.Errorf("serving: %w", err)
	}
}

// helper to recognize benign close errors
func isServerClosed(err error) bool {
	// If your server returns net.ErrClosed or a custom sentinel, normalize here.
	return errors.Is(err, net.ErrClosed) // TODO: replace with a real error sentinel if needed
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

		// newMaster now runs the handler directly and returns its final error.
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
