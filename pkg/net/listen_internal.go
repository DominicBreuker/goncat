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

// listenDependencies holds injectable dependencies for testing.
type listenDependencies struct {
	listenAndServeTCP func(context.Context, string, time.Duration, transport.Handler, *log.Logger, *config.Dependencies) error
	listenAndServeWS  func(context.Context, string, time.Duration, transport.Handler, *log.Logger) error
	listenAndServeWSS func(context.Context, string, time.Duration, transport.Handler, *log.Logger) error
	listenAndServeUDP func(context.Context, string, time.Duration, transport.Handler, *log.Logger) error
}

// Real implementations for production use.
func realListenAndServeTCP(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger, deps *config.Dependencies) error {
	return tcp.ListenAndServe(ctx, addr, timeout, handler, logger, deps)
}

func realListenAndServeWS(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
	return ws.ListenAndServeWS(ctx, addr, timeout, handler, logger)
}

func realListenAndServeWSS(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
	return ws.ListenAndServeWSS(ctx, addr, timeout, handler, logger)
}

func realListenAndServeUDP(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler, logger *log.Logger) error {
	return udp.ListenAndServe(ctx, addr, timeout, handler, logger)
}

// serveWithTransport calls the appropriate transport's ListenAndServe function.
func serveWithTransport(ctx context.Context, cfg *config.Shared, handler transport.Handler, deps *listenDependencies) error {
	addr := cfg.Host + ":" + fmt.Sprint(cfg.Port)

	// Wrap handler with TLS if requested
	wrappedHandler, err := wrapHandlerWithTLS(handler, cfg)
	if err != nil {
		return fmt.Errorf("preparing handler: %w", err)
	}

	switch cfg.Protocol {
	case config.ProtoWS:
		return deps.listenAndServeWS(ctx, addr, cfg.Timeout, wrappedHandler, cfg.Logger)
	case config.ProtoWSS:
		return deps.listenAndServeWSS(ctx, addr, cfg.Timeout, wrappedHandler, cfg.Logger)
	case config.ProtoUDP:
		// UDP/QUIC handles transport-level TLS internally
		// Application-level TLS (--ssl) is applied via handler wrapper
		return deps.listenAndServeUDP(ctx, addr, cfg.Timeout, wrappedHandler, cfg.Logger)
	default:
		// Default to TCP
		return deps.listenAndServeTCP(ctx, addr, cfg.Timeout, wrappedHandler, cfg.Logger, cfg.Deps)
	}
}

// wrapHandlerWithTLS wraps the handler with TLS if SSL is enabled.
func wrapHandlerWithTLS(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
	if !cfg.SSL {
		return handler, nil
	}

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
