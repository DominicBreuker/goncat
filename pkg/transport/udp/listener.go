package udp

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"

	quic "github.com/quic-go/quic-go"
)

// Platform-specific socket option implementations are in:
// - listener_unix.go for Unix-like systems (Linux, macOS, BSD)
// - listener_windows.go for Windows

// ListenAndServe creates a QUIC listener over UDP and serves connections.
// The function blocks until the context is cancelled or an error occurs.
// Up to 100 concurrent connections are allowed; additional connections are rejected.
//
// The timeout parameter controls stream accept operations (from the --timeout flag).
// QUIC MaxIdleTimeout is set to 3x timeout (minimum 30s) for connection keep-alive.
//
// QUIC provides built-in TLS 1.3 encryption at the transport layer.
// An ephemeral certificate is generated for the transport layer TLS.
// Application-level TLS (--ssl, --key) is handled separately by the caller.
//
// QUIC streams require an init byte to activate - this is handled internally.
func ListenAndServe(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
	// Create UDP listener with SO_REUSEADDR
	udpConn, err := createUDPListener(ctx, addr)
	if err != nil {
		return err
	}
	defer udpConn.Close()

	// Generate ephemeral TLS config for QUIC transport
	tlsConfig, err := generateQUICServerTLSConfig()
	if err != nil {
		return err
	}

	// Create QUIC listener
	quicListener, err := createQUICListener(udpConn, tlsConfig, timeout)
	if err != nil {
		return err
	}
	defer quicListener.Close()

	// Serve connections with semaphore, passing timeout through
	sem := createConnectionSemaphore(100)
	return serveQUICConnections(ctx, quicListener, handler, logger, sem, timeout)
}

// createUDPListener creates a UDP socket with SO_REUSEADDR for rapid port reuse.
func createUDPListener(ctx context.Context, addr string) (*net.UDPConn, error) {
	lc := &net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var sockOptErr error
			err := c.Control(func(fd uintptr) {
				sockOptErr = setSockoptReuseAddr(fd)
			})
			if err != nil {
				return err
			}
			return sockOptErr
		},
	}

	packetConn, err := lc.ListenPacket(ctx, "udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}

	udpConn, ok := packetConn.(*net.UDPConn)
	if !ok {
		packetConn.Close()
		return nil, fmt.Errorf("expected *net.UDPConn, got %T", packetConn)
	}

	return udpConn, nil
}

// generateQUICServerTLSConfig generates an ephemeral TLS configuration for QUIC server.
func generateQUICServerTLSConfig() (*tls.Config, error) {
	// Generate ephemeral certificate
	key := rand.Text()
	_, cert, err := crypto.GenerateCertificates(key)
	if err != nil {
		return nil, fmt.Errorf("generate certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"goncat-quic"},
	}

	return tlsConfig, nil
}

// createQUICListener creates a QUIC listener with proper timeout configuration.
func createQUICListener(udpConn *net.UDPConn, tlsConfig *tls.Config, timeout time.Duration) (*quic.Listener, error) {
	// Configure QUIC timeouts
	maxIdleTimeout := timeout * 3
	if maxIdleTimeout < 30*time.Second {
		maxIdleTimeout = 30 * time.Second
	}

	quicConfig := &quic.Config{
		MaxIdleTimeout:  maxIdleTimeout,
		KeepAlivePeriod: maxIdleTimeout / 3,
	}

	// Create QUIC transport and listener
	tr := &quic.Transport{Conn: udpConn}
	quicListener, err := tr.Listen(tlsConfig, quicConfig)
	if err != nil {
		return nil, fmt.Errorf("quic listen: %w", err)
	}

	return quicListener, nil
}

// createConnectionSemaphore creates a buffered channel to limit concurrent connections.
func createConnectionSemaphore(capacity int) chan struct{} {
	sem := make(chan struct{}, capacity)
	for i := 0; i < capacity; i++ {
		sem <- struct{}{}
	}
	return sem
}

// serveQUICConnections accepts and handles QUIC connections.
// It respects context cancellation and propagates the timeout to stream operations.
func serveQUICConnections(ctx context.Context, quicListener *quic.Listener, handler transport.Handler, logger *log.Logger, sem chan struct{}, timeout time.Duration) error {
	// Run accept loop in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- acceptQUICLoop(ctx, quicListener, handler, logger, sem, timeout)
	}()

	// Wait for either context cancellation or accept loop error
	select {
	case <-ctx.Done():
		// Context cancelled - close listener
		_ = quicListener.Close()
		err := <-errCh
		// Treat closure as graceful shutdown
		if err == nil || isClosedError(err) {
			return nil
		}
		return fmt.Errorf("serving after cancellation: %w", err)

	case err := <-errCh:
		// Accept loop exited on its own
		if err == nil || isClosedError(err) {
			return nil
		}
		return fmt.Errorf("accept quic: %w", err)
	}
}

// acceptQUICLoop accepts QUIC connections and spawns handlers.
// It uses the caller's context for Accept operations to support graceful cancellation.
func acceptQUICLoop(ctx context.Context, quicListener *quic.Listener, handler transport.Handler, logger *log.Logger, sem chan struct{}, timeout time.Duration) error {
	for {
		// Accept QUIC connection using caller's context
		conn, err := quicListener.Accept(ctx)
		if err != nil {
			// Check if context was cancelled
			if ctx.Err() != nil {
				return nil // Graceful shutdown
			}
			// Check for clean shutdown
			if isClosedError(err) {
				return nil
			}
			return err
		}

		// Try to acquire a slot
		select {
		case <-sem:
			go handleQUICConnection(ctx, conn, handler, logger, sem, timeout)
		default:
			// All slots busy
			_ = conn.CloseWithError(0x42, "server busy")
		}
	}
}

// handleQUICConnection processes a single QUIC connection.
func handleQUICConnection(ctx context.Context, conn *quic.Conn, handler transport.Handler, logger *log.Logger, sem chan struct{}, timeout time.Duration) {
	defer func() {
		sem <- struct{}{} // Release slot
	}()
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorMsg("Handler panic: %v\n", r)
		}
	}()

	// Accept stream and read init byte
	stream, err := acceptStreamAndActivate(ctx, conn, timeout)
	if err != nil {
		_ = conn.CloseWithError(0, err.Error())
		return
	}

	// Wrap stream in net.Conn adapter
	streamConn := NewStreamConn(conn, stream, conn.LocalAddr(), conn.RemoteAddr())
	defer streamConn.Close()

	if err := handler(streamConn); err != nil {
		logger.ErrorMsg("Handling connection: %s\n", err)
	}
}

// acceptStreamAndActivate accepts a stream and reads the init byte.
// The client sends an init byte to activate the stream immediately.
// Uses the provided timeout from the --timeout flag for stream acceptance.
func acceptStreamAndActivate(ctx context.Context, conn *quic.Conn, timeout time.Duration) (*quic.Stream, error) {
	// Accept first bidirectional stream with user-provided timeout
	acceptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stream, err := conn.AcceptStream(acceptCtx)
	if err != nil {
		return nil, fmt.Errorf("no stream")
	}

	// Read and discard init byte
	initByte := make([]byte, 1)
	_, err = stream.Read(initByte)
	if err != nil {
		return nil, fmt.Errorf("failed to read init byte")
	}

	return stream, nil
}

// isClosedError checks if an error indicates a closed connection.
func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "server closed" ||
		strings.Contains(errStr, "use of closed network connection")
}
