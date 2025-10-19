package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"sync"
)

// uses interfaces/factories from internal.go (DI for testing)

// SlaveConnect connects to a remote master and follows its instructions as a slave.
func SlaveConnect(ctx context.Context, cfg *config.Shared) error {
	return slaveConnect(ctx, cfg, realClientFactory(), realSlaveFactory())
}

// slaveConnect is the internal implementation that accepts injected dependencies for testing.
func slaveConnect(
	ctx context.Context,
	cfg *config.Shared,
	newClient clientFactory,
	newSlave slaveFactory,
) error {
	c := newClient(ctx, cfg)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	var closeOnce sync.Once
	defer closeOnce.Do(func() { _ = c.Close() })

	go func() {
		<-ctx.Done()
		closeOnce.Do(func() { _ = c.Close() })
	}()

	// let user know we established a new connection successfully
	remoteAddr := c.GetConnection().RemoteAddr().String()
	log.InfoMsg("New connection from %s\n", remoteAddr)
	defer log.InfoMsg("Connection to %s closed\n", remoteAddr)

	h, err := newSlave(ctx, cfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("creating slave: %w", err)
	}
	defer h.Close()

	if err := h.Handle(); err != nil {
		return fmt.Errorf("handling: %w", err)
	}

	return nil
}
