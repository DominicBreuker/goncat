package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"fmt"
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
		return fmt.Errorf("connecting: %s", err)
	}
	defer c.Close()

	// Ensure client connection is closed when the parent context is cancelled.
	go func() {
		<-ctx.Done()
		_ = c.Close()
	}()

	h, err := newSlave(ctx, cfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("slave.New(): %s", err)
	}
	defer h.Close()

	if err := h.Handle(); err != nil {
		return fmt.Errorf("handling: %s", err)
	}

	return nil
}
