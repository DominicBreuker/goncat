package server

import (
	"crypto/tls"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"net"
)

// Idea: add WebSocket as connection method based on https://pkg.go.dev/github.com/coder/websocket#NetConn

// Server ...
type Server struct {
	cfg *config.Shared

	l net.Listener
}

// New ...
func New(cfg *config.Shared) *Server {
	return &Server{
		cfg: cfg,
	}
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

	if s.cfg.SSL {
		caCert, cert, err := crypto.GenerateCertificates(s.cfg.GetKey())
		if err != nil {
			return fmt.Errorf("crypto.GenerateCertificates(%s): %s", s.cfg.GetKey(), err)
		}

		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if s.cfg.GetKey() != "" {
			cfg.ClientAuth = tls.RequireAndVerifyClientCert
			cfg.ClientCAs = caCert
		}

		s.l, err = tls.Listen("tcp", addr, cfg)
		if err != nil {
			return fmt.Errorf("listen(tcp, %s): %s", addr, err)
		}
	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
		}

		s.l, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return fmt.Errorf("listen(tcp, %s): %s", addr, err)
		}
	}

	log.InfoMsg("Listening on %s\n", addr)

	return nil
}

// Accept ...
func (s *Server) Accept() (net.Conn, error) {
	conn, err := s.l.Accept()
	if err != nil {
		log.ErrorMsg("Accept(): %s\n", err)
	}

	log.InfoMsg("New connection from %s\n", conn.RemoteAddr())

	return conn, nil
}
