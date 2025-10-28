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

// Dial establishes a QUIC connection over UDP and returns a stream as net.Conn.
// QUIC provides built-in TLS 1.3 encryption at the transport layer.
//
// The timeout parameter controls the QUIC MaxIdleTimeout (3x timeout, minimum 30s).
// An ephemeral certificate is generated for the transport layer TLS.
// Application-level TLS (--ssl, --key) is handled separately by the caller.
//
// QUIC streams require an init byte to activate - this is handled internally.
func Dial(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	// Resolve address
	udpAddr, err := resolveUDPAddress(addr)
	if err != nil {
		return nil, err
	}

	// Create UDP socket
	udpConn, err := createUDPSocket()
	if err != nil {
		return nil, err
	}

	// Generate ephemeral TLS config for QUIC transport
	tlsConfig, err := generateQUICTLSConfig()
	if err != nil {
		udpConn.Close()
		return nil, err
	}

	// Establish QUIC connection
	quicConn, err := dialQUIC(ctx, udpConn, udpAddr, tlsConfig, timeout)
	if err != nil {
		udpConn.Close()
		return nil, err
	}

	// Open stream and activate it
	stream, err := openAndActivateStream(quicConn)
	if err != nil {
		quicConn.CloseWithError(0, "failed to open stream")
		udpConn.Close()
		return nil, err
	}

	return NewStreamConn(quicConn, stream, quicConn.LocalAddr(), quicConn.RemoteAddr()), nil
}

// resolveUDPAddress parses and resolves a UDP address string.
func resolveUDPAddress(addr string) (*net.UDPAddr, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve udp addr: %w", err)
	}
	return udpAddr, nil
}

// createUDPSocket creates a UDP socket with a system-assigned local port.
func createUDPSocket() (*net.UDPConn, error) {
	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}
	return udpConn, nil
}

// generateQUICTLSConfig generates an ephemeral TLS configuration for QUIC transport.
func generateQUICTLSConfig() (*tls.Config, error) {
	// Generate ephemeral certificate
	key := rand.Text()
	_, cert, err := crypto.GenerateCertificates(key)
	if err != nil {
		return nil, fmt.Errorf("generate certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ServerName:         "goncat",
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{"goncat-quic"},
		InsecureSkipVerify: true, // Accept any server cert for transport layer
	}

	return tlsConfig, nil
}

// dialQUIC establishes a QUIC connection.
func dialQUIC(ctx context.Context, udpConn *net.UDPConn, udpAddr *net.UDPAddr, tlsConfig *tls.Config, timeout time.Duration) (*quic.Conn, error) {
	// Configure QUIC timeouts
	// MaxIdleTimeout should be longer than control operations to avoid premature closure
	maxIdleTimeout := timeout * 3
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
	conn, err := tr.Dial(ctx, udpAddr, tlsConfig, quicConfig)
	if err != nil {
		return nil, fmt.Errorf("quic dial: %w", err)
	}

	return conn, nil
}

// openAndActivateStream opens a bidirectional QUIC stream and activates it.
// QUIC streams are lazy-initialized - we write an init byte to force activation.
// The server will read and discard this byte.
func openAndActivateStream(conn *quic.Conn) (*quic.Stream, error) {
	// Open bidirectional stream
	stream, err := conn.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}

	// Write init byte to activate the stream
	// This ensures the server's AcceptStream() completes immediately
	_, err = stream.Write([]byte{0})
	if err != nil {
		return nil, fmt.Errorf("write init byte: %w", err)
	}

	return stream, nil
}
