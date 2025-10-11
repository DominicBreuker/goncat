package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/server"
	"fmt"
	"net"
)

// SlaveListen starts a server that listens for incoming master connections
// and follows their instructions as a slave.
func SlaveListen(ctx context.Context, cfg *config.Shared) error {
	s, err := server.New(ctx, cfg, makeSlaveHandler(ctx, cfg))
	if err != nil {
		return fmt.Errorf("server.New(): %s", err)
	}

	if err := s.Serve(); err != nil {
		return fmt.Errorf("serving: %s", err)
	}

	return nil
}

func makeSlaveHandler(ctx context.Context, cfg *config.Shared) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
		defer conn.Close()

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
