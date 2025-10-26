package udp

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"dominicbreuker/goncat/pkg/crypto"
	quic "github.com/quic-go/quic-go"
)

// Dialer implements transport.Dialer for UDP with QUIC.
type Dialer struct {
	remoteAddr *net.UDPAddr
	timeout    time.Duration
}

// NewDialer creates a UDP+QUIC dialer for the specified remote address.
// Parameters:
// - addr: Remote address to connect to (e.g., "192.168.1.100:12345")
// - timeout: MaxIdleTimeout for QUIC connection
//
// Note: QUIC mandates TLS 1.3. An ephemeral certificate is generated internally.
// Application-level TLS (--ssl, --key) is handled separately by the caller,
// similar to how WebSocket (wss) handles transport-level TLS separately from app-level TLS.
func NewDialer(addr string, timeout time.Duration) (*Dialer, error) {
	// Parse UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve udp addr: %w", err)
	}

	return &Dialer{
		remoteAddr: udpAddr,
		timeout:    timeout,
	}, nil
}

// Dial establishes a QUIC connection and opens a bidirectional stream.
// QUIC requires TLS 1.3, so an ephemeral certificate is generated for the transport layer.
// This is separate from any application-level TLS (--ssl, --key) that may be applied by the caller.
func (d *Dialer) Dial(ctx context.Context) (net.Conn, error) {
	// Create UDP socket (use system-assigned local port)
	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}

	// Generate ephemeral TLS config for QUIC transport layer.
	// Similar to how WebSocket (wss) handles its transport TLS internally.
	// Application-level TLS (if --ssl/--key is set) will be applied on top.
	key := rand.Text()
	_, cert, err := crypto.GenerateCertificates(key)
	if err != nil {
		udpConn.Close()
		return nil, fmt.Errorf("generate certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ServerName:         "goncat",
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{"goncat-quic"},
		InsecureSkipVerify: true, // Accept any server cert for transport layer
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
	conn, err := tr.Dial(ctx, d.remoteAddr, tlsConfig, quicConfig)
	if err != nil {
		udpConn.Close()
		return nil, fmt.Errorf("quic dial: %w", err)
	}

	// Open bidirectional stream
	stream, err := conn.OpenStream()
	if err != nil {
		conn.CloseWithError(0, "failed to open stream")
		udpConn.Close()
		return nil, fmt.Errorf("open stream: %w", err)
	}

	// CRITICAL: QUIC streams are lazy-initialized and not transmitted until data is written.
	// We must write an initialization byte to force the stream to be sent to the peer.
	// The server will read and discard this byte. This ensures the server's AcceptStream()
	// call will complete immediately rather than blocking indefinitely.
	_, err = stream.Write([]byte{0})
	if err != nil {
		conn.CloseWithError(0, "failed to write init byte")
		udpConn.Close()
		return nil, fmt.Errorf("write init byte: %w", err)
	}

	// Wrap in net.Conn adapter
	return NewStreamConn(stream, conn.LocalAddr(), conn.RemoteAddr()), nil
}
