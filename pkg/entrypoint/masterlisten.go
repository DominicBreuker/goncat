// Package entrypoint provides entry functions for the four operation modes of goncat.
// These functions encapsulate the logic for starting servers/clients and handlers,
// separating it from CLI argument parsing.
package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"net"
)

// uses interfaces/factories from internal.go (DI for testing)

// MasterListen starts a server that listens for incoming slave connections
// and controls them as a master.
func MasterListen(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterListen(ctx, cfg, mCfg, realServerFactory(), makeMasterHandler)
}

// masterListen is the internal implementation that accepts injected dependencies for testing.
func masterListen(
	ctx context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	newServer serverFactory,
	makeHandler func(context.Context, *config.Shared, *config.Master) func(net.Conn) error,
) error {
	s, err := newServer(ctx, cfg, makeHandler(ctx, cfg, mCfg))
	if err != nil {
		return fmt.Errorf("server.New(): %s", err)
	}

	// Always close the server when this function returns.
	defer func() {
		_ = s.Close()
	}()

	// Ensure the server is closed when the context is cancelled so Serve() can return.
	go func() {
		<-ctx.Done()
		// best-effort close to unblock Accept/Serve; log any error for diagnostics
		if err := s.Close(); err != nil {
			log.InfoMsg("master entrypoint: error closing server on context done: %s", err)
		}
	}()

	if err := s.Serve(); err != nil {
		return fmt.Errorf("serving: %s", err)
	}

	return nil
}

func makeMasterHandler(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
		defer conn.Close()

		// Close the active connection when the parent context is cancelled so
		// per-connection handlers (which may block on reads) can exit promptly.
		go func() {
			<-ctx.Done()
			conn.Close()
		}()

		mst, err := master.New(ctx, cfg, mCfg, conn)
		if err != nil {
			return fmt.Errorf("master.New(): %s", err)
		}
		defer mst.Close()

		if err := mst.Handle(); err != nil {
			return fmt.Errorf("handle: %s", err)
		}

		return nil
	}
}
