package udp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	quic "github.com/quic-go/quic-go"
)

// Dialer implements transport.Dialer for UDP with QUIC.
type Dialer struct {
	remoteAddr *net.UDPAddr
	timeout    time.Duration
	tlsConfig  *tls.Config
}

// NewDialer creates a UDP+QUIC dialer for the specified remote address.
// Parameters:
// - addr: Remote address to connect to (e.g., "192.168.1.100:12345")
// - timeout: MaxIdleTimeout for QUIC connection
// - tlsConfig: TLS configuration (required by QUIC, must include ServerName and use TLS 1.3+)
func NewDialer(addr string, timeout time.Duration, tlsConfig *tls.Config) (*Dialer, error) {
	// Parse UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve udp addr: %w", err)
	}

	// Ensure TLS 1.3 (required by QUIC)
	if tlsConfig.MinVersion < tls.VersionTLS13 {
		tlsConfig.MinVersion = tls.VersionTLS13
	}

	return &Dialer{
		remoteAddr: udpAddr,
		timeout:    timeout,
		tlsConfig:  tlsConfig,
	}, nil
}

// Dial establishes a QUIC connection and opens a bidirectional stream.
func (d *Dialer) Dial(ctx context.Context) (net.Conn, error) {
	// Create UDP socket (use system-assigned local port)
	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}

	// Configure QUIC
	// MaxIdleTimeout should be longer than the timeout used for control operations
	// to avoid premature connection closure. Use at least 30 seconds or 3x the timeout.
	maxIdleTimeout := d.timeout * 3
	if maxIdleTimeout < 30*time.Second {
		maxIdleTimeout = 30 * time.Second
	}
	quicConfig := &quic.Config{
		MaxIdleTimeout:  maxIdleTimeout,
		KeepAlivePeriod: maxIdleTimeout / 3,
	}

	// Create QUIC transport
	tr := &quic.Transport{Conn: udpConn}

	// Dial QUIC connection
	conn, err := tr.Dial(ctx, d.remoteAddr, d.tlsConfig, quicConfig)
	if err != nil {
		udpConn.Close()
		return nil, fmt.Errorf("quic dial: %w", err)
	}

	// Open bidirectional stream
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(0, "failed to open stream")
		udpConn.Close()
		return nil, fmt.Errorf("open stream: %w", err)
	}

	// Wrap in net.Conn adapter
	return NewStreamConn(stream, conn.LocalAddr(), conn.RemoteAddr()), nil
}
