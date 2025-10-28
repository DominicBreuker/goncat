package net

import (
	"context"
	"crypto/tls"
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

// Real implementations for production use.
func realCreateListener(ctx context.Context, cfg *config.Shared) (transport.Listener, error) {
	addr := cfg.Host + ":" + fmt.Sprint(cfg.Port)

	var (
		listener transport.Listener
		err      error
	)

	switch cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		listener, err = ws.NewListener(ctx, addr, cfg.Protocol == config.ProtoWSS)
		if err != nil {
			cfg.Logger.VerboseMsg("Failed to create WebSocket listener: %v", err)
			return nil, fmt.Errorf("create WebSocket listener: %w", err)
		}

	case config.ProtoUDP:
		// UDP/QUIC handles transport-level TLS internally (like WebSocket wss)
		// Application-level TLS (--ssl) is applied via handler wrapper
		listener, err = udp.NewListener(ctx, addr, cfg.Timeout)
		if err != nil {
			cfg.Logger.VerboseMsg("Failed to create UDP listener: %v", err)
			return nil, fmt.Errorf("create UDP listener: %w", err)
		}

	default:
		// Default to TCP
		listener, err = tcp.NewListener(addr, cfg.Deps)
		if err != nil {
			cfg.Logger.VerboseMsg("Failed to create TCP listener: %v", err)
			return nil, fmt.Errorf("create TCP listener: %w", err)
		}
	}

	return listener, nil
}

func realWrapWithTLS(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
	// Build TLS configuration
	tlsConfig, err := buildServerTLSConfig(cfg.GetKey(), cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("building server TLS config: %w", err)
	}

	// Return wrapped handler
	return func(conn net.Conn) error {
		cfg.Logger.VerboseMsg("Starting TLS handshake with %s", conn.RemoteAddr())

		// Wrap connection with TLS
		tlsConn := tls.Server(conn, tlsConfig)

		// Perform TLS handshake with timeout
		if err := performServerTLSHandshake(tlsConn, cfg.Timeout, cfg.Logger); err != nil {
			_ = tlsConn.Close()
			return fmt.Errorf("TLS handshake: %w", err)
		}

		cfg.Logger.VerboseMsg("TLS handshake completed with %s", conn.RemoteAddr())

		// Call original handler with TLS connection
		return handler(tlsConn)
	}, nil
}

// buildServerTLSConfig creates the TLS configuration for the server.
func buildServerTLSConfig(key string, logger *log.Logger) (*tls.Config, error) {
	logger.VerboseMsg("Generating TLS certificates for server")

	caCert, cert, err := crypto.GenerateCertificates(key)
	if err != nil {
		return nil, fmt.Errorf("generate certificates: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	if key != "" {
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		cfg.ClientCAs = caCert
		logger.VerboseMsg("TLS mutual authentication enabled")
	}

	return cfg, nil
}

// performServerTLSHandshake performs the server-side TLS handshake with timeout handling.
// CRITICAL: Sets deadline before handshake, clears it immediately after success.
func performServerTLSHandshake(tlsConn *tls.Conn, timeout time.Duration, logger *log.Logger) error {
	// Set handshake deadline to avoid blocking indefinitely
	if timeout > 0 {
		_ = tlsConn.SetDeadline(time.Now().Add(timeout))
	}

	err := tlsConn.Handshake()

	// Clear deadline immediately after handshake completes (success or failure)
	// This is critical - lingering deadlines can kill healthy connections later
	if timeout > 0 {
		_ = tlsConn.SetDeadline(time.Time{})
	}

	if err != nil {
		logger.VerboseMsg("TLS server handshake failed: %v", err)
		return err
	}

	logger.VerboseMsg("TLS server handshake completed successfully")
	return nil
}
