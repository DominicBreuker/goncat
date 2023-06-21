package server

import (
	"crypto/tls"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/exec"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
	"net"
)

// Server ...
type Server struct {
	cfg config.Config
}

// New ...
func New(cfg config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Serve ...
func (s *Server) Serve() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var l net.Listener
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

		l, err = tls.Listen("tcp", addr, cfg)
		if err != nil {
			return fmt.Errorf("listen(tcp, %s): %s", addr, err)
		}
	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
		}

		l, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return fmt.Errorf("listen(tcp, %s): %s", addr, err)
		}
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.ErrorMsg("Accept(): %s\n", err)
		}

		s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	log.InfoMsg("New connection from %s\n", conn.RemoteAddr())
	defer log.InfoMsg("Connection from %s lost\n", conn.RemoteAddr())

	connCtl, connData, err := mux.AcceptChannels(conn)
	if err != nil {
		log.ErrorMsg("Handling %s: mux.AcceptChannels(conn): %s\n", conn.RemoteAddr(), err)
		return
	}
	defer connCtl.Close()
	defer connData.Close()
	defer conn.Close()

	if s.cfg.Pty {
		if err := s.handleWithPTY(connCtl, connData); err != nil {
			log.ErrorMsg("Handling %s with PTY: %s\n", conn.RemoteAddr(), err)
		}
	} else {
		if err := s.handlePlain(connData); err != nil {
			log.ErrorMsg("Handling %s: %s\n", conn.RemoteAddr(), err)
		}
	}
}

func (s *Server) handleWithPTY(connCtl, connData net.Conn) error {
	if s.cfg.LogFile != "" {
		var err error
		connData, err = log.NewLoggedConn(connData, s.cfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", s.cfg.LogFile, err)
		}
	}

	if s.cfg.Exec != "" {
		if err := exec.RunWithPTY(connCtl, connData, s.cfg.Exec, s.cfg.Verbose); err != nil {
			return fmt.Errorf("exec.RunWithPTY(...): %s", err)
		}
	} else {
		if err := terminal.PipeWithPTY(connCtl, connData, s.cfg.Verbose); err != nil {
			return fmt.Errorf("terminal.PipeWithPTY(...): %s", err)
		}
	}

	return nil
}

func (s *Server) handlePlain(conn net.Conn) error {
	if s.cfg.LogFile != "" {
		var err error
		conn, err = log.NewLoggedConn(conn, s.cfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", s.cfg.LogFile, err)
		}
	}

	if s.cfg.Exec != "" {
		if err := exec.Run(conn, s.cfg.Exec); err != nil {
			return fmt.Errorf("exec.Run(conn, %s): %s", s.cfg.Exec, err)
		}
	} else {
		terminal.Pipe(conn, s.cfg.Verbose)
	}

	return nil
}
