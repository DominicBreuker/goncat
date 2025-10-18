package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"dominicbreuker/goncat/pkg/handler/slave"
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

// handlerInterface defines the interface for a handler that can handle connections.
type handlerInterface interface {
	Handle() error
	Close() error
}

// masterFactory is a function type for creating master handlers.
type masterFactory func(context.Context, *config.Shared, *config.Master, net.Conn) (handlerInterface, error)

// realMasterFactory returns the actual master factory used in production.
func realMasterFactory() masterFactory {
	return func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (handlerInterface, error) {
		return master.New(ctx, cfg, mCfg, conn)
	}
}

// slaveFactory is a function type for creating slave handlers.
type slaveFactory func(context.Context, *config.Shared, net.Conn) (handlerInterface, error)

// realSlaveFactory returns the actual slave factory used in production.
func realSlaveFactory() slaveFactory {
	return func(ctx context.Context, cfg *config.Shared, conn net.Conn) (handlerInterface, error) {
		return slave.New(ctx, cfg, conn)
	}
}
