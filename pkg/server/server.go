// Package server provides a network server implementation with support for
// multiple transport protocols (TCP, WebSocket) and optional TLS encryption.
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"dominicbreuker/goncat/pkg/transport/tcp"
	"dominicbreuker/goncat/pkg/transport/ws"
	"fmt"
	"net"
	"sync"
	"time"
)

// dependencies holds the injectable dependencies for testing.
type dependencies struct {
	newTCPListener func(string, *config.Dependencies) (transport.Listener, error)
	newWSListener  func(context.Context, string, bool) (transport.Listener, error)
	certGenerator  func(string) (*x509.CertPool, tls.Certificate, error)
}

// Server manages a network listener and handles incoming connections
// with optional TLS encryption and mutual authentication.
type Server struct {
	ctx  context.Context
	cfg  *config.Shared
	deps *dependencies

	mu     sync.Mutex
	l      transport.Listener
	handle transport.Handler
}

// New creates a new Server with the given context, configuration, and connection handler.
// If SSL is enabled in the configuration, connections are wrapped with TLS using generated certificates.
// If a key is configured, mutual TLS authentication is required.
func New(ctx context.Context, cfg *config.Shared, handle transport.Handler) (*Server, error) {
	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			return tcp.NewListener(addr, deps)
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			return ws.NewListener(ctx, addr, secure)
		},
		certGenerator: crypto.GenerateCertificates,
	}
	return newServer(ctx, cfg, handle, deps)
}

// newServer is the internal implementation that accepts injected dependencies for testing.
func newServer(ctx context.Context, cfg *config.Shared, handle transport.Handler, deps *dependencies) (*Server, error) {
	s := &Server{
		ctx:  ctx,
		cfg:  cfg,
		deps: deps,
	}

	if cfg.SSL {
		caCert, cert, err := deps.certGenerator(s.cfg.GetKey())
		if err != nil {
			return nil, fmt.Errorf("crypto.GenerateCertificates(%s): %s", s.cfg.GetKey(), err)
		}

		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		}
		if cfg.GetKey() != "" {
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
			tlsCfg.ClientCAs = caCert
			tlsCfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
				return customVerifier(caCert, rawCerts)
			}
		}

		s.handle = func(conn net.Conn) error {
			tlsConn := tls.Server(conn, tlsCfg)

			// set a handshake deadline to avoid blocking forever
			_ = tlsConn.SetDeadline(time.Now().Add(cfg.Timeout))
			if err := tlsConn.Handshake(); err != nil {
				// ensure connection closed on handshake failure
				_ = tlsConn.Close()
				return fmt.Errorf("tls handshake: %s", err)
			}
			// clear deadline after handshake
			_ = tlsConn.SetDeadline(time.Time{})

			return handle(tlsConn)
		}
	} else {
		s.handle = handle
	}

	return s, nil
}

// Close stops the server's listener if it's running.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.l != nil {
		return s.l.Close()
	}
	return nil
}

// Serve starts the server and begins accepting connections.
// The server listens on the configured address using the configured transport protocol.
// This function blocks until an error occurs or the server is closed.
func (s *Server) Serve() error {
	addr := format.Addr(s.cfg.Host, s.cfg.Port)

	var l transport.Listener
	var err error
	switch s.cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		l, err = s.deps.newWSListener(s.ctx, addr, s.cfg.Protocol == config.ProtoWSS)
	default:
		l, err = s.deps.newTCPListener(addr, s.cfg.Deps)
	}
	if err != nil {
		return fmt.Errorf("tcp.New(%s): %s", addr, err)
	}

	s.mu.Lock()
	s.l = l
	s.mu.Unlock()

	log.InfoMsg("Listening on %s\n", addr)

	return s.l.Serve(s.handle)
}

// customVerifier verifies the certificate but cares only about the root certificate, not SANs
func customVerifier(caCert *x509.CertPool, rawCerts [][]byte) error {
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
