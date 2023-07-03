package server

import (
	"crypto/tls"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"net"
)

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

// Serve ...
func (s *Server) Serve() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

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

//func (s *Server) Handle(conn net.Conn) {
//	defer log.InfoMsg("Connection from %s lost\n", conn.RemoteAddr())
//
//	connCtl, connData, err := mux.AcceptChannels(conn)
//	if err != nil {
//		log.ErrorMsg("Handling %s: mux.AcceptChannels(conn): %s\n", conn.RemoteAddr(), err)
//		return
//	}
//	defer connCtl.Close()
//	defer connData.Close()
//	defer conn.Close()
//
//	if s.cfg.Pty {
//		if err := s.handleWithPTY(connCtl, connData); err != nil {
//			log.ErrorMsg("Handling %s with PTY: %s\n", conn.RemoteAddr(), err)
//		}
//	} else {
//		if err := s.handlePlain(connData); err != nil {
//			log.ErrorMsg("Handling %s: %s\n", conn.RemoteAddr(), err)
//		}
//	}
//}
//
