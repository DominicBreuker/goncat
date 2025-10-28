package net

import (
	"context"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/transport"
)

// ListenAndServe creates a listener on the configured address and serves
// incoming connections using the provided handler function.
// It supports TCP, WebSocket, and UDP protocols with optional TLS encryption.
// The function blocks until the context is cancelled or an error occurs.
//
// The handler function is called for each accepted connection. It should
// handle the connection and return when done. The connection will be
// closed after the handler returns.
//
// All cleanup, timeout management, and resource lifecycle is handled
// internally. Callers only need to provide the handler logic.
func ListenAndServe(ctx context.Context, cfg *config.Shared, handler transport.Handler) error {
	deps := &listenDependencies{
		listenAndServeTCP: realListenAndServeTCP,
		listenAndServeWS:  realListenAndServeWS,
		listenAndServeWSS: realListenAndServeWSS,
		listenAndServeUDP: realListenAndServeUDP,
	}
	return listenAndServe(ctx, cfg, handler, deps)
}

// listenAndServe is the internal implementation.
func listenAndServe(ctx context.Context, cfg *config.Shared, handler transport.Handler, deps *listenDependencies) error {
	addr := format.Addr(cfg.Host, cfg.Port)
	cfg.Logger.VerboseMsg("Creating listener for protocol %s at %s", cfg.Protocol, addr)

	cfg.Logger.InfoMsg("Listening on %s\n", addr)
	cfg.Logger.VerboseMsg("Server ready, waiting for connections")

	// Serve with transport-specific function
	return serveWithTransport(ctx, cfg, handler, deps)
}
