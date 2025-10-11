// Package entrypoint provides entry functions for the four operation modes of goncat.
// These functions encapsulate the logic for starting servers/clients and handlers,
// separating it from CLI argument parsing.
package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/server"
	"fmt"
	"net"
)

// MasterListen starts a server that listens for incoming slave connections
// and controls them as a master.
func MasterListen(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	s, err := server.New(ctx, cfg, makeMasterHandler(ctx, cfg, mCfg))
	if err != nil {
		return fmt.Errorf("server.New(): %s", err)
	}

	if err := s.Serve(); err != nil {
		return fmt.Errorf("serving: %s", err)
	}
	defer s.Close()

	return nil
}

func makeMasterHandler(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
		defer conn.Close()

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
