package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"dominicbreuker/goncat/pkg/transport/tcp"
	"dominicbreuker/goncat/pkg/transport/udp"
	"dominicbreuker/goncat/pkg/transport/ws"
)

type Server struct {
	ctx    context.Context
	cfg    *config.Shared
	l      transport.Listener
	handle transport.Handler
}

func New(ctx context.Context, cfg *config.Shared, handle transport.Handler) (*Server, error) {
	s := &Server{ctx: ctx, cfg: cfg}

	// Wrap handler with application-level TLS if --ssl is set.
	// This applies to all transports: TCP, WS, WSS, and UDP.
	// WebSocket (wss) and UDP already have transport-level TLS, but app-level TLS is separate.
	if cfg.SSL {
		cfg.Logger.VerboseMsg("Generating TLS certificates for server")
		caCert, cert, err := crypto.GenerateCertificates(cfg.GetKey())
		if err != nil {
			return nil, fmt.Errorf("generate certificates: %w", err)
		}

		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		}
		if cfg.GetKey() != "" {
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
			tlsCfg.ClientCAs = caCert
			cfg.Logger.VerboseMsg("TLS mutual authentication enabled")
		}

		s.handle = func(conn net.Conn) error {
			cfg.Logger.VerboseMsg("Starting TLS handshake with %s", conn.RemoteAddr())
			tlsConn := tls.Server(conn, tlsCfg)
			if cfg.Timeout > 0 {
				_ = tlsConn.SetDeadline(time.Now().Add(cfg.Timeout))
				defer func() { _ = tlsConn.SetDeadline(time.Time{}) }()
			}
			if err := tlsConn.Handshake(); err != nil {
				cfg.Logger.VerboseMsg("TLS handshake failed with %s: %v", conn.RemoteAddr(), err)
				_ = tlsConn.Close()
				return fmt.Errorf("tls handshake: %w", err)
			}
			cfg.Logger.VerboseMsg("TLS handshake completed with %s", conn.RemoteAddr())
			return handle(tlsConn)
		}
	} else {
		s.handle = handle
	}

	return s, nil
}

func (s *Server) Close() error {
	if s.l != nil {
		return s.l.Close()
	}
	return nil
}

func (s *Server) Serve() error {
	addr := format.Addr(s.cfg.Host, s.cfg.Port)

	s.cfg.Logger.VerboseMsg("Creating listener for protocol %s at %s", s.cfg.Protocol, addr)

	var (
		l   transport.Listener
		err error
	)
	switch s.cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		l, err = ws.NewListener(s.ctx, addr, s.cfg.Protocol == config.ProtoWSS)
	case config.ProtoUDP:
		// UDP/QUIC handles transport-level TLS internally (like WebSocket wss)
		// Application-level TLS (--ssl) is applied via handler wrapper in New()
		l, err = udp.NewListener(s.ctx, addr, s.cfg.Timeout)
	default:
		l, err = tcp.NewListener(addr, s.cfg.Deps)
	}
	if err != nil {
		s.cfg.Logger.VerboseMsg("Failed to create listener: %v", err)
		return fmt.Errorf("new listener %s (%s): %w", addr, s.cfg.Protocol, err)
	}
	s.l = l

	log.InfoMsg("Listening on %s\n", addr)
	s.cfg.Logger.VerboseMsg("Server ready, waiting for connections")
	return s.l.Serve(s.handle)
}
