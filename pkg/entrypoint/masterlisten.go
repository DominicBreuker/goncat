// Package entrypoint provides entry functions for the four operation modes of goncat.
// These functions encapsulate the logic for starting servers/clients and handlers,
// separating it from CLI argument parsing.
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

// MasterListen starts a server that listens for incoming slave connections
// and controls them as a master.
func MasterListen(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterListen(ctx, cfg, mCfg, realServerFactory(), realMasterFactory())
}

// masterListen is the internal implementation that accepts injected dependencies for testing.
func masterListen(
	ctx context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	newServer serverFactory,
	newMaster masterFactory,
) error {
	s, err := newServer(ctx, cfg, makeMasterHandler(ctx, cfg, mCfg, newMaster))
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

func makeMasterHandler(ctx context.Context, cfg *config.Shared, mCfg *config.Master, newMaster masterFactory) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		// let user know about connection status
		remoteAddr := conn.RemoteAddr().String()
		log.InfoMsg("New connection from %s\n", remoteAddr)
		defer log.InfoMsg("Connection to %s closed\n", remoteAddr)

		var connOnce sync.Once
		defer connOnce.Do(func() { _ = conn.Close() })

		go func() {
			<-ctx.Done()
			connOnce.Do(func() { _ = conn.Close() })
		}()

		mst, err := newMaster(ctx, cfg, mCfg, conn)
		if err != nil {
			return fmt.Errorf("master.New(): %s", err)
		}
		defer mst.Close()

		if err := mst.Handle(); err != nil {
			return fmt.Errorf("handling: %w", err)
		}

		return nil
	}
}
