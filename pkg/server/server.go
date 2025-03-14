package server

import (
	"context"
	"crypto/tls"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"dominicbreuker/goncat/pkg/transport/tcp"
	"dominicbreuker/goncat/pkg/transport/ws"
	"fmt"
	"net"
)

// Server ...
type Server struct {
	ctx context.Context
	cfg *config.Shared

	l      transport.Listener
	handle transport.Handler
}

// New ...
func New(ctx context.Context, cfg *config.Shared, handle transport.Handler) (*Server, error) {
	s := &Server{
		ctx: ctx,
		cfg: cfg,
	}

	if cfg.SSL {
		caCert, cert, err := crypto.GenerateCertificates(s.cfg.GetKey())
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
		}

		s.handle = func(conn net.Conn) error {
			tlsConn := tls.Server(conn, tlsCfg)
			tlsConn.Handshake()
			return handle(tlsConn)
		}
	} else {
		s.handle = handle
	}

	return s, nil
}

// Close ...
func (s *Server) Close() error {
	if s.l != nil {
		return s.l.Close()
	}
	return nil
}

// Serve ...
func (s *Server) Serve() error {
	addr := format.Addr(s.cfg.Host, s.cfg.Port)

	var err error
	switch s.cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		s.l, err = ws.NewListener(s.ctx, addr, s.cfg.Protocol == config.ProtoWSS)
	default:
		s.l, err = tcp.NewListener(addr)
	}
	if err != nil {
		return fmt.Errorf("tcp.New(%s): %s", addr, err)
	}

	log.InfoMsg("Listening on %s\n", addr)

	return s.l.Serve(s.handle)
}
