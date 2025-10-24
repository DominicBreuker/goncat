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

	if cfg.SSL {
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
		}

		s.handle = func(conn net.Conn) error {
			tlsConn := tls.Server(conn, tlsCfg)
			if cfg.Timeout > 0 {
				_ = tlsConn.SetDeadline(time.Now().Add(cfg.Timeout))
				defer func() { _ = tlsConn.SetDeadline(time.Time{}) }()
			}
			if err := tlsConn.Handshake(); err != nil {
				_ = tlsConn.Close()
				return fmt.Errorf("tls handshake: %w", err)
			}
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

	var (
		l   transport.Listener
		err error
	)
	switch s.cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		l, err = ws.NewListener(s.ctx, addr, s.cfg.Protocol == config.ProtoWSS)
	case config.ProtoUDP:
		l, err = udp.NewListener(addr, s.cfg.Deps)
	default:
		l, err = tcp.NewListener(addr, s.cfg.Deps)
	}
	if err != nil {
		return fmt.Errorf("new listener %s (%s): %w", addr, s.cfg.Protocol, err)
	}
	s.l = l

	log.InfoMsg("Listening on %s\n", addr)
	return s.l.Serve(s.handle)
}
