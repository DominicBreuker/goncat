package tcp

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ListenAndServe creates a TCP listener and serves connections until context is cancelled.
// Up to 100 concurrent connections are allowed; additional connections are rejected.
// The function blocks until the context is cancelled or an error occurs.
// All cleanup and resource management is handled internally.
//
// The handler function is called for each accepted connection in a separate goroutine.
// The connection is automatically closed when the handler returns.
func ListenAndServe(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger, deps *config.Dependencies) error {
	// Create listener
	listener, err := createListener(addr, deps)
	if err != nil {
		return err
	}
	defer listener.Close()

	// Create semaphore for connection limiting
	sem := createConnectionSemaphore(100)

	// Serve connections with context handling
	return serveConnections(ctx, listener, handler, logger, sem)
}

// createListener creates a TCP listener on the specified address.
func createListener(addr string, deps *config.Dependencies) (net.Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}

	listenerFn := config.GetTCPListenerFunc(deps)

	nl, err := listenerFn("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("listen(tcp, %s): %w", addr, err)
	}

	return nl, nil
}

// createConnectionSemaphore creates a buffered channel to limit concurrent connections.
func createConnectionSemaphore(capacity int) chan struct{} {
	sem := make(chan struct{}, capacity)
	// Initially allow 'capacity' active connections
	for i := 0; i < capacity; i++ {
		sem <- struct{}{}
	}
	return sem
}

// serveConnections accepts and handles connections until context is cancelled.
func serveConnections(ctx context.Context, listener net.Listener, handler transport.Handler, logger *log.Logger, sem chan struct{}) error {
	// Channel for accept loop errors
	errCh := make(chan error, 1)

	// Run accept loop in goroutine
	go func() {
		errCh <- acceptLoop(listener, handler, logger, sem)
	}()

	// Wait for either context cancellation or accept loop error
	select {
	case <-ctx.Done():
		// Context cancelled - close listener and wait for accept loop to exit
		_ = listener.Close()
		err := <-errCh
		// Treat closure due to context cancellation as graceful
		if err == nil || isListenerClosed(err) {
			return nil
		}
		return fmt.Errorf("serving after cancellation: %w", err)

	case err := <-errCh:
		// Accept loop exited on its own
		if err == nil || isListenerClosed(err) {
			return nil
		}
		return fmt.Errorf("serving: %w", err)
	}
}

// acceptLoop accepts connections and spawns handlers.
func acceptLoop(listener net.Listener, handler transport.Handler, logger *log.Logger, sem chan struct{}) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Treat listener closed as clean shutdown
			if isListenerClosed(err) {
				return nil
			}
			// Retry on timeouts with a short backoff
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("Accept(): %w", err)
		}

		// Try to acquire a slot
		select {
		case <-sem:
			// Acquired slot - handle connection
			go handleConnection(conn, handler, logger, sem)
		default:
			// All slots busy - reject connection
			_ = conn.Close()
		}
	}
}

// handleConnection processes a single connection.
func handleConnection(conn net.Conn, handler transport.Handler, logger *log.Logger, sem chan struct{}) {
	// Always release slot and close connection
	defer func() {
		_ = conn.Close()
		sem <- struct{}{} // Release slot
	}()

	// Prevent panic from leaking the slot
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorMsg("Handler panic: %v\n", r)
		}
	}()

	if err := handler(conn); err != nil {
		logger.ErrorMsg("Handling connection: %s\n", err)
	}
}

// isListenerClosed checks if an error indicates a closed listener.
func isListenerClosed(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, net.ErrClosed) ||
		strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "listener closed")
}
