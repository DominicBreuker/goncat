package ws

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// ListenAndServeWS creates a WebSocket listener (plain HTTP) and serves connections.
// The function blocks until the context is cancelled or an error occurs.
// Up to 100 concurrent connections are allowed; additional connections receive HTTP 503.
//
// The handler function is called for each accepted WebSocket connection.
// All cleanup and resource management is handled internally.
func ListenAndServeWS(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
	return listenAndServeWebSocket(ctx, addr, timeout, handler, logger, false)
}

// ListenAndServeWSS creates a WebSocket Secure listener (HTTPS/TLS) and serves connections.
// The function blocks until the context is cancelled or an error occurs.
// Up to 100 concurrent connections are allowed; additional connections receive HTTP 503.
//
// TLS is enabled at the transport layer with an ephemeral self-signed certificate.
// The handler function is called for each accepted WebSocket connection.
// All cleanup and resource management is handled internally.
func ListenAndServeWSS(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
	return listenAndServeWebSocket(ctx, addr, timeout, handler, logger, true)
}

// listenAndServeWebSocket is the internal implementation for both ws and wss.
func listenAndServeWebSocket(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger, useTLS bool) error {
	// Create network listener
	listener, err := createNetListener(addr, useTLS)
	if err != nil {
		return err
	}
	defer listener.Close()

	// Create semaphore for connection limiting
	sem := createConnectionSemaphore(100)

	// Create HTTP server
	server := createHTTPServer(ctx, handler, logger, sem)

	// Serve with context handling
	return serveWithContext(ctx, server, listener)
}

// createNetListener creates a TCP listener with optional TLS.
func createNetListener(addr string, useTLS bool) (net.Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}

	var nl net.Listener
	nl, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.ListenTCP(tcp, %s): %w", tcpAddr.String(), err)
	}

	if useTLS {
		nl, err = wrapWithTLS(nl)
		if err != nil {
			return nil, fmt.Errorf("wrap with TLS: %w", err)
		}
	}

	return nl, nil
}

// wrapWithTLS wraps a listener with TLS using an ephemeral certificate.
func wrapWithTLS(nl net.Listener) (net.Listener, error) {
	// Generate ephemeral certificate for transport-level TLS
	key := rand.Text()
	_, cert, err := crypto.GenerateCertificates(key)
	if err != nil {
		return nil, fmt.Errorf("crypto.GenerateCertificates(%s): %w", key, err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	return tls.NewListener(nl, tlsCfg), nil
}

// createConnectionSemaphore creates a buffered channel to limit concurrent connections.
func createConnectionSemaphore(capacity int) chan struct{} {
	sem := make(chan struct{}, capacity)
	for i := 0; i < capacity; i++ {
		sem <- struct{}{}
	}
	return sem
}

// createHTTPServer creates an HTTP server that upgrades connections to WebSocket.
func createHTTPServer(ctx context.Context, handler transport.Handler, logger *log.Logger, sem chan struct{}) *http.Server {
	return &http.Server{
		Handler: createWebSocketHandler(ctx, handler, logger, sem),

		// Timeouts for long-lived tunnel connections
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0,                // Unlimited after headers
		WriteTimeout:      0,                // No write timeout
		IdleTimeout:       60 * time.Second, // Standard idle timeout
	}
}

// createWebSocketHandler creates an HTTP handler that upgrades to WebSocket.
func createWebSocketHandler(ctx context.Context, handler transport.Handler, logger *log.Logger, sem chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to acquire a slot
		select {
		case <-sem:
			// Acquired - handle connection
			defer func() { sem <- struct{}{} }()

			handleWebSocketUpgrade(ctx, w, r, handler, logger)

		default:
			// All slots busy - reject with 503
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	}
}

// handleWebSocketUpgrade upgrades the HTTP connection to WebSocket and handles it.
func handleWebSocketUpgrade(ctx context.Context, w http.ResponseWriter, r *http.Request, handler transport.Handler, logger *log.Logger) {
	// Accept WebSocket upgrade
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"bin"},
	})
	if err != nil {
		logger.ErrorMsg("websocket.Accept(): %s\n", err)
		return
	}

	// Wrap as net.Conn
	conn := websocket.NetConn(ctx, c, websocket.MessageBinary)
	logger.InfoMsg("New WS connection from %s\n", conn.RemoteAddr())

	// Ensure connection is closed
	defer func() { _ = conn.Close() }()

	// Prevent panic from leaking resources
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorMsg("Handler panic: %v\n", r)
		}
	}()

	// Handle the connection
	if err := handler(conn); err != nil {
		logger.ErrorMsg("handle websocket.NetConn: %s\n", err)
	}
}

// serveWithContext runs the HTTP server with context cancellation support.
func serveWithContext(ctx context.Context, server *http.Server, listener net.Listener) error {
	// Run server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	// Wait for either context cancellation or server error
	select {
	case <-ctx.Done():
		// Context cancelled - close listener
		_ = listener.Close()
		err := <-errCh
		// Treat closure as graceful shutdown
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serving after cancellation: %w", err)

	case err := <-errCh:
		// Server exited on its own
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http.Server.Serve(): %w", err)
	}
}
