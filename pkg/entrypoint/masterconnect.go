package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"fmt"
)

// MasterConnect connects to a remote slave and controls it as a master.
func MasterConnect(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	c := client.New(ctx, cfg)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("connecting: %s", err)
	}
	defer c.Close()

	// Ensure client connection is closed when the parent context is cancelled.
	go func() {
		<-ctx.Done()
		_ = c.Close()
		if conn := c.GetConnection(); conn != nil {
			conn.Close()
		}
	}()

	h, err := master.New(ctx, cfg, mCfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("master.New(): %s", err)
	}
	defer h.Close()

	if err := h.Handle(); err != nil {
		return fmt.Errorf("handling: %s", err)
	}

	return nil
}
