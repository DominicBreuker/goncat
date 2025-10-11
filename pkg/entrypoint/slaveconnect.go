package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"fmt"
)

// SlaveConnect connects to a remote master and follows its instructions as a slave.
func SlaveConnect(ctx context.Context, cfg *config.Shared) error {
	c := client.New(ctx, cfg)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("connecting: %s", err)
	}
	defer c.Close()

	h, err := slave.New(ctx, cfg, c.GetConnection())
	if err != nil {
		return fmt.Errorf("slave.New(): %s", err)
	}
	defer h.Close()

	if err := h.Handle(); err != nil {
		return fmt.Errorf("handling: %s", err)
	}

	return nil
}
