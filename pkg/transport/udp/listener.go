package udp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	quic "github.com/quic-go/quic-go"
)

// Platform-specific socket option implementations are in:
// - listener_unix.go for Unix-like systems (Linux, macOS, BSD)
// - listener_windows.go for Windows

// Listener implements transport.Listener for UDP with QUIC.
// It ensures only one connection is handled at a time via a semaphore.
type Listener struct {
	udpConn      *net.UDPConn
	quicListener *quic.Listener
	sem          chan struct{} // capacity 1 -> allows a single active handler
}

// NewListener creates a UDP+QUIC listener on the specified address.
// Parameters:
// - ctx: Context for lifecycle management
// - addr: Address to bind (e.g., "0.0.0.0:12345" or ":12345")
// - timeout: MaxIdleTimeout for QUIC connections
// - tlsConfig: TLS configuration (required by QUIC, must use TLS 1.3+)
func NewListener(ctx context.Context, addr string, timeout time.Duration, tlsConfig *tls.Config) (*Listener, error) {
	// Create UDP socket with SO_REUSEADDR for rapid port reuse (important for E2E tests)
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

	// Configure QUIC
	// MaxIdleTimeout should be longer than the timeout used for control operations
	// to avoid premature connection closure. Use at least 30 seconds or 3x the timeout.
	maxIdleTimeout := timeout * 3
	if maxIdleTimeout < 30*time.Second {
		maxIdleTimeout = 30 * time.Second
	}
	quicConfig := &quic.Config{
		MaxIdleTimeout:  maxIdleTimeout,
		KeepAlivePeriod: maxIdleTimeout / 3, // Keep alive more frequently than idle timeout
	}

	// Ensure TLS 1.3 (required by QUIC)
	if tlsConfig.MinVersion < tls.VersionTLS13 {
		tlsConfig.MinVersion = tls.VersionTLS13
	}

	// Create QUIC transport and listener
	tr := &quic.Transport{Conn: udpConn}
	quicListener, err := tr.Listen(tlsConfig, quicConfig)
	if err != nil {
		udpConn.Close()
		return nil, fmt.Errorf("quic listen: %w", err)
	}

	l := &Listener{
		udpConn:      udpConn,
		quicListener: quicListener,
		sem:          make(chan struct{}, 1),
	}
	l.sem <- struct{}{} // initially allow one connection

	return l, nil
}

// Serve accepts QUIC connections and handles them using the provided handler.
// Only one connection is handled at a time; additional connections are rejected.
func (l *Listener) Serve(handle transport.Handler) error {
	for {
		// Accept QUIC connection
		conn, err := l.quicListener.Accept(context.Background())
		if err != nil {
			// Check for clean shutdown
			if isClosedError(err) {
				return nil
			}
			return fmt.Errorf("accept quic: %w", err)
		}

		// Try to acquire single slot
		select {
		case <-l.sem:
			go l.handleConnection(conn, handle)
		default:
			// Already handling one connection
			_ = conn.CloseWithError(0x42, "server busy")
		}
	}
}

// handleConnection processes a single QUIC connection.
func (l *Listener) handleConnection(conn *quic.Conn, handle transport.Handler) {
	defer func() {
		l.sem <- struct{}{} // release slot
	}()
	defer func() {
		if r := recover(); r != nil {
			log.ErrorMsg("Handler panic: %v\n", r)
		}
	}()

	// Accept first bidirectional stream with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		_ = conn.CloseWithError(0, "no stream")
		return
	}

	// Read and discard the initialization byte that was sent by the client to activate the stream.
	// QUIC streams are lazy-initialized - the client must write data before the stream is
	// transmitted and visible to AcceptStream(). This single byte ensures immediate connection
	// establishment without blocking.
	initByte := make([]byte, 1)
	_, err = stream.Read(initByte)
	if err != nil {
		_ = conn.CloseWithError(0, "failed to read init byte")
		return
	}

	// Wrap stream in net.Conn adapter
	streamConn := NewStreamConn(stream, conn.LocalAddr(), conn.RemoteAddr())
	defer streamConn.Close()

	if err := handle(streamConn); err != nil {
		log.ErrorMsg("Handling connection: %s\n", err)
	}
}

// Close stops the listener.
func (l *Listener) Close() error {
	if err := l.quicListener.Close(); err != nil {
		return err
	}
	return l.udpConn.Close()
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
