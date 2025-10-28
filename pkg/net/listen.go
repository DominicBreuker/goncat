package net

import (
"context"
"errors"
"fmt"
"net"
"sync"

"dominicbreuker/goncat/pkg/config"
"dominicbreuker/goncat/pkg/format"
"dominicbreuker/goncat/pkg/log"
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
createListener: realCreateListener,
wrapWithTLS:    realWrapWithTLS,
}
return listenAndServe(ctx, cfg, handler, deps)
}

// listenDependencies holds injectable dependencies for testing.
type listenDependencies struct {
createListener func(context.Context, *config.Shared) (transport.Listener, error)
wrapWithTLS    func(transport.Handler, *config.Shared) (transport.Handler, error)
}

// listenAndServe is the internal implementation.
func listenAndServe(ctx context.Context, cfg *config.Shared, handler transport.Handler, deps *listenDependencies) error {
addr := format.Addr(cfg.Host, cfg.Port)

cfg.Logger.VerboseMsg("Creating listener for protocol %s at %s", cfg.Protocol, addr)

// Step 1: Create the appropriate listener for the protocol
listener, err := deps.createListener(ctx, cfg)
if err != nil {
return fmt.Errorf("creating listener: %w", err)
}

var closeOnce sync.Once
closeListener := func() {
closeOnce.Do(func() {
_ = listener.Close()
})
}
defer closeListener()

// Step 2: Wrap handler with TLS if requested
wrappedHandler := handler
if cfg.SSL {
wrappedHandler, err = deps.wrapWithTLS(handler, cfg)
if err != nil {
return fmt.Errorf("wrapping handler with TLS: %w", err)
}
}

// Step 3: Serve connections
log.InfoMsg("Listening on %s\n", addr)
cfg.Logger.VerboseMsg("Server ready, waiting for connections")

// Run serve in goroutine to handle context cancellation
errCh := make(chan error, 1)
go func() {
errCh <- listener.Serve(wrappedHandler)
}()

// Wait for either context cancellation or serve error
select {
case <-ctx.Done():
cfg.Logger.VerboseMsg("Context cancelled, shutting down server")
closeListener()
err := <-errCh
// Treat closure due to context cancellation as graceful
if err == nil || isServerClosed(err) || errors.Is(err, context.Canceled) {
return nil
}
return fmt.Errorf("serving after cancellation: %w", err)

case err := <-errCh:
// Serve exited on its own
if err == nil || isServerClosed(err) {
return nil
}
return fmt.Errorf("serving: %w", err)
}
}

// isServerClosed recognizes benign close errors.
func isServerClosed(err error) bool {
return errors.Is(err, net.ErrClosed)
}
