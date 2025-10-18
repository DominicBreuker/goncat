package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"net"
)

// uses interfaces/factories from internal.go (DI for testing)

// SlaveListen starts a server that listens for incoming master connections
// and follows their instructions as a slave.
func SlaveListen(ctx context.Context, cfg *config.Shared) error {
	return slaveListen(ctx, cfg, realServerFactory(), makeSlaveHandler)
}

// slaveListen is the internal implementation that accepts injected dependencies for testing.
func slaveListen(
	ctx context.Context,
	cfg *config.Shared,
	newServer serverFactory,
	makeHandler func(context.Context, *config.Shared) func(net.Conn) error,
) error {
	s, err := newServer(ctx, cfg, makeHandler(ctx, cfg))
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
			log.InfoMsg("slave entrypoint: error closing server on context done: %s", err)
		}
	}()

	if err := s.Serve(); err != nil {
		return fmt.Errorf("serving: %s", err)
	}

	return nil
}

func makeSlaveHandler(ctx context.Context, cfg *config.Shared) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
		defer conn.Close()

		// Close the active connection when the parent context is cancelled so
		// per-connection handlers (which may block on reads) can exit promptly.
		go func() {
			<-ctx.Done()
			conn.Close()
		}()

		slv, err := slave.New(ctx, cfg, conn)
		if err != nil {
			return fmt.Errorf("slave.New(): %s", err)
		}
		defer slv.Close()

		if err := slv.Handle(); err != nil {
			return fmt.Errorf("handle: %s", err)
		}

		return nil
	}
}
