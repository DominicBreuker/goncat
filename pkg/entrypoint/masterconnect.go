package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"fmt"
)

// uses interfaces/factories from internal.go (DI for testing)

// MasterConnect connects to a remote slave and controls it as a master.
func MasterConnect(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterConnect(ctx, cfg, mCfg, realClientFactory(), realMasterFactory())
}

// masterConnect is the internal implementation that accepts injected dependencies for testing.
func masterConnect(
	ctx context.Context,
	cfg *config.Shared,
	mCfg *config.Master,
	newClient clientFactory,
	newMaster masterFactory,
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

	h, err := newMaster(ctx, cfg, mCfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("master.New(): %s", err)
	}
	defer h.Close()

	if err := h.Handle(); err != nil {
		return fmt.Errorf("handling: %s", err)
	}

	return nil
}
