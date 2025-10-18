package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"fmt"
	"net"
)

// clientFactory is a function type for creating clients.
type clientFactory func(context.Context, *config.Shared) clientInterface

// clientInterface defines the interface for a client that can connect and provide a connection.
type clientInterface interface {
	Connect() error
	Close() error
	GetConnection() net.Conn
}

// handlerInterface defines the interface for a handler that can handle connections.
type handlerInterface interface {
	Handle() error
	Close() error
}

// masterFactory is a function type for creating master handlers.
type masterFactory func(context.Context, *config.Shared, *config.Master, net.Conn) (handlerInterface, error)

// MasterConnect connects to a remote slave and controls it as a master.
func MasterConnect(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
	return masterConnect(ctx, cfg, mCfg, func(ctx context.Context, cfg *config.Shared) clientInterface {
		return client.New(ctx, cfg)
	}, func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (handlerInterface, error) {
		return master.New(ctx, cfg, mCfg, conn)
	})
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
