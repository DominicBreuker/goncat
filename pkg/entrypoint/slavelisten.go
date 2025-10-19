package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"net"
	"sync"
)

// uses interfaces/factories from internal.go (DI for testing)

// SlaveListen starts a server that listens for incoming master connections
// and follows their instructions as a slave.
func SlaveListen(ctx context.Context, cfg *config.Shared) error {
	return slaveListen(ctx, cfg, realServerFactory(), realSlaveFactory())
}

// slaveListen is the internal implementation that accepts injected dependencies for testing.
func slaveListen(
	ctx context.Context,
	cfg *config.Shared,
	newServer serverFactory,
	newSlave slaveFactory,
) error {
	s, err := newServer(ctx, cfg, makeSlaveHandler(ctx, cfg, newSlave))
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	var closeOnce sync.Once
	defer closeOnce.Do(func() { _ = s.Close() })

	go func() {
		<-ctx.Done()
		closeOnce.Do(func() { _ = s.Close() })
	}()

	if err := s.Serve(); err != nil {
		return fmt.Errorf("serving: %w", err)
	}

	return nil
}

func makeSlaveHandler(ctx context.Context, cfg *config.Shared, newSlave slaveFactory) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())

		var connOnce sync.Once
		defer connOnce.Do(func() { _ = conn.Close() })

		go func() {
			<-ctx.Done()
			connOnce.Do(func() { _ = conn.Close() })
		}()

		slv, err := newSlave(ctx, cfg, conn)
		if err != nil {
			return fmt.Errorf("slave.New(): %s", err)
		}
		defer slv.Close()

		if err := slv.Handle(); err != nil {
			return fmt.Errorf("handling: %w", err)
		}

		return nil
	}
}
