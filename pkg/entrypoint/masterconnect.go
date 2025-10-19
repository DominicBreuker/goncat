package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"fmt"
	"sync"
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
		return fmt.Errorf("connecting: %w", err)
	}
	var closeOnce sync.Once
	defer closeOnce.Do(func() { _ = c.Close() })

	go func() {
		<-ctx.Done()
		closeOnce.Do(func() { _ = c.Close() })
	}()

	h, err := newMaster(ctx, cfg, mCfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("creating master: %w", err)
	}
	defer h.Close()

	if err := h.Handle(); err != nil {
		return fmt.Errorf("handling: %w", err)
	}

	return nil
}
