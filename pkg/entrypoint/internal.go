package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/server"
	"dominicbreuker/goncat/pkg/transport"
	"net"
)

// serverInterface defines the interface for a server that can serve and be closed.
type serverInterface interface {
	Serve() error
	Close() error
}

// serverFactory is a function type for creating servers.
type serverFactory func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error)

// realServerFactory returns the actual server factory used in production.
func realServerFactory() serverFactory {
	return func(ctx context.Context, cfg *config.Shared, handle transport.Handler) (serverInterface, error) {
		return server.New(ctx, cfg, handle)
	}
}

// clientInterface defines the interface for a client that can connect and provide a connection.
type clientInterface interface {
	Connect() error
	Close() error
	GetConnection() net.Conn
}

// clientFactory is a function type for creating clients.
type clientFactory func(context.Context, *config.Shared) clientInterface

// realClientFactory returns the actual client factory used in production.
func realClientFactory() clientFactory {
	return func(ctx context.Context, cfg *config.Shared) clientInterface {
		return client.New(ctx, cfg)
	}
}

// masterHandler is the master handler function.
type masterHandler func(context.Context, *config.Shared, *config.Master, net.Conn) error

// slaveHandler is the slave handler function signature (runs the handler directly
// and returns its final error). This mirrors masterHandler.
type slaveHandler func(context.Context, *config.Shared, net.Conn) error
