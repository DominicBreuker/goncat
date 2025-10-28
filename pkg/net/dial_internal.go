package net

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"dominicbreuker/goncat/pkg/transport/tcp"
	"dominicbreuker/goncat/pkg/transport/udp"
	"dominicbreuker/goncat/pkg/transport/ws"
)

// dialDependencies holds injectable dependencies for testing.
type dialDependencies struct {
	newTCPDialer func(string, *config.Dependencies) (transport.Dialer, error)
	newWSDialer  func(context.Context, string, config.Protocol) transport.Dialer
	newUDPDialer func(string, time.Duration) (transport.Dialer, error)
}

// Real implementations for production use.
func realNewTCPDialer(addr string, deps *config.Dependencies) (transport.Dialer, error) {
	return tcp.NewDialer(addr, deps)
}

func realNewWSDialer(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
	return ws.NewDialer(ctx, addr, proto)
}

func realNewUDPDialer(addr string, timeout time.Duration) (transport.Dialer, error) {
	return udp.NewDialer(addr, timeout)
}

// createDialer creates the appropriate transport dialer based on protocol.
func createDialer(ctx context.Context, cfg *config.Shared, deps *dialDependencies) (transport.Dialer, error) {
	addr := cfg.Host + ":" + fmt.Sprint(cfg.Port)

	switch cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		return deps.newWSDialer(ctx, addr, cfg.Protocol), nil

	case config.ProtoUDP:
		// UDP/QUIC handles transport-level TLS internally (like WebSocket wss)
		// Application-level TLS (--ssl) will be applied after connection if needed
		dialer, err := deps.newUDPDialer(addr, cfg.Timeout)
		if err != nil {
			cfg.Logger.VerboseMsg("Failed to create UDP dialer: %v", err)
			return nil, fmt.Errorf("create UDP dialer: %w", err)
		}
		return dialer, nil

	default:
		// Default to TCP
		dialer, err := deps.newTCPDialer(addr, cfg.Deps)
		if err != nil {
			cfg.Logger.VerboseMsg("Failed to create TCP dialer: %v", err)
			return nil, fmt.Errorf("create TCP dialer: %w", err)
		}
		return dialer, nil
	}
}

// establishConnection dials using the provided dialer with timeout handling.
// CRITICAL: Sets deadline before dial, clears it immediately after success.
func establishConnection(ctx context.Context, dialer transport.Dialer, cfg *config.Shared) (net.Conn, error) {
	conn, err := dialer.Dial(ctx)
	if err != nil {
		cfg.Logger.VerboseMsg("Connection failed: %v", err)
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	// Clear any deadlines set by the dialer to ensure healthy connection
	// This is critical - lingering deadlines can kill healthy connections later
	if cfg.Timeout > 0 {
		_ = conn.SetDeadline(time.Time{})
	}

	cfg.Logger.VerboseMsg("Connection established")
	return conn, nil
}

// upgradeTLS wraps the connection with TLS, handling timeouts properly.
// CRITICAL: Sets deadline before handshake, clears it immediately after success.
func upgradeTLS(conn net.Conn, cfg *config.Shared) (net.Conn, error) {
	// Build TLS configuration
	tlsConfig, err := buildTLSConfig(cfg.GetKey(), cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("building TLS config: %w", err)
	}

	// Wrap connection with TLS
	tlsConn := tls.Client(conn, tlsConfig)

	// Perform TLS handshake with timeout
	if err := performTLSHandshake(tlsConn, cfg.Timeout, cfg.Logger); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}

	return tlsConn, nil
}

// buildTLSConfig creates the TLS configuration for the client.
func buildTLSConfig(key string, logger *log.Logger) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // custom verification below
	}

	if key != "" {
		logger.VerboseMsg("Generating TLS client certificates for mutual authentication")
		caCert, cert, err := crypto.GenerateCertificates(key)
		if err != nil {
			return nil, fmt.Errorf("generate certificates: %w", err)
		}

		cfg.Certificates = []tls.Certificate{cert}
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return verifyPeerCertificate(caCert, rawCerts)
		}
		logger.VerboseMsg("TLS mutual authentication configured")
	}

	return cfg, nil
}

// performTLSHandshake performs the TLS handshake with proper timeout handling.
// CRITICAL: Sets deadline before handshake, clears it immediately after success.
func performTLSHandshake(tlsConn *tls.Conn, timeout time.Duration, logger *log.Logger) error {
	// Set handshake deadline to avoid blocking indefinitely
	if timeout > 0 {
		_ = tlsConn.SetDeadline(time.Now().Add(timeout))
	}

	logger.VerboseMsg("Starting TLS client handshake")
	err := tlsConn.Handshake()

	// Clear deadline immediately after handshake completes (success or failure)
	// This is critical - lingering deadlines can kill healthy connections later
	if timeout > 0 {
		_ = tlsConn.SetDeadline(time.Time{})
	}

	if err != nil {
		logger.VerboseMsg("TLS client handshake failed: %v", err)
		return err
	}

	logger.VerboseMsg("TLS client handshake completed successfully")
	return nil
}

// verifyPeerCertificate validates server certificate against CA pool.
// It cares only about the root certificate, not SANs.
func verifyPeerCertificate(caCert *x509.CertPool, rawCerts [][]byte) error {
	if len(rawCerts) != 1 {
		return fmt.Errorf("unexpected number of raw certs: %d", len(rawCerts))
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	if _, err := cert.Verify(x509.VerifyOptions{
		Roots: caCert,
	}); err != nil {
		return fmt.Errorf("verify certificate: %w", err)
	}

	return nil
}
