package net

import (
	"context"
	"errors"
	"fmt"
	"net"

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

	// Create listener
	listener, err := deps.createListener(ctx, cfg)
	if err != nil {
		return fmt.Errorf("creating listener: %w", err)
	}
	defer listener.Close()

	// Wrap handler with TLS if requested
	wrappedHandler, err := prepareHandler(handler, cfg, deps)
	if err != nil {
		return fmt.Errorf("preparing handler: %w", err)
	}

	// Serve with context handling
	log.InfoMsg("Listening on %s\n", addr)
	cfg.Logger.VerboseMsg("Server ready, waiting for connections")

	return serveWithContext(ctx, listener, wrappedHandler, cfg)
}

// prepareHandler wraps the handler with TLS if SSL is enabled.
func prepareHandler(handler transport.Handler, cfg *config.Shared, deps *listenDependencies) (transport.Handler, error) {
	if !cfg.SSL {
		return handler, nil
	}

	wrappedHandler, err := deps.wrapWithTLS(handler, cfg)
	if err != nil {
		return nil, fmt.Errorf("wrapping with TLS: %w", err)
	}

	return wrappedHandler, nil
}

// serveWithContext runs listener.Serve in a goroutine and handles context cancellation.
func serveWithContext(ctx context.Context, listener transport.Listener, handler transport.Handler, cfg *config.Shared) error {
	// Run serve in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- listener.Serve(handler)
	}()

	// Wait for either context cancellation or serve error
	select {
	case <-ctx.Done():
		return handleContextCancellation(listener, errCh, cfg)
	case err := <-errCh:
		return handleServeCompletion(err)
	}
}

// handleContextCancellation closes the listener and waits for serve to exit.
func handleContextCancellation(listener transport.Listener, errCh <-chan error, cfg *config.Shared) error {
	cfg.Logger.VerboseMsg("Context cancelled, shutting down server")
	_ = listener.Close()

	err := <-errCh
	// Treat closure due to context cancellation as graceful
	if err == nil || isServerClosed(err) || errors.Is(err, context.Canceled) {
		return nil
	}
	return fmt.Errorf("serving after cancellation: %w", err)
}

// handleServeCompletion checks if serve exited gracefully or with an error.
func handleServeCompletion(err error) error {
	if err == nil || isServerClosed(err) {
		return nil
	}
	return fmt.Errorf("serving: %w", err)
}

// isServerClosed recognizes benign close errors.
func isServerClosed(err error) bool {
	return errors.Is(err, net.ErrClosed)
}
