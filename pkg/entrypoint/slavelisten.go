package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"errors"
	"fmt"
	"net"
	"sync"
)

// uses interfaces/factories from internal.go (DI for testing)

func SlaveListen(ctx context.Context, cfg *config.Shared) error {
	return slaveListen(ctx, cfg, realServerFactory(), slave.Handle)
}

func slaveListen(
	parent context.Context,
	cfg *config.Shared,
	newServer serverFactory,
	handle slaveHandler,
) error {
	// child context we can cancel on return
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	s, err := newServer(ctx, cfg, makeSlaveHandler(ctx, cfg, handle))
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	var closeOnce sync.Once
	closeServer := func() { closeOnce.Do(func() { _ = s.Close() }) }
	defer closeServer()

	// run Serve in a goroutine and wait deterministically
	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve() }()

	select {
	case <-ctx.Done():
		closeServer()
		err := <-errCh
		if err == nil || isServerClosed(err) || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("serving after cancel: %w", err)

	case err := <-errCh:
		if err == nil || isServerClosed(err) {
			return nil
		}
		return fmt.Errorf("serving: %w", err)
	}
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
			if err == nil || errors.Is(err, context.Canceled) /* || errors.Is(err, net.ErrClosed) */ {
				return nil
			}
			return fmt.Errorf("handling after cancel: %w", err)

		case err := <-errCh:
			if err == nil /* || errors.Is(err, net.ErrClosed) */ {
				return nil
			}
			return fmt.Errorf("handling: %w", err)
		}
	}
}
