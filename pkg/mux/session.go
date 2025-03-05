package mux

import (
	"io"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

// Session ...
type Session struct {
	mux *yamux.Session

	ctlClientToServer net.Conn
	ctlServerToClient net.Conn
}

// Close ...
func (s *Session) Close() error {
	if s.ctlClientToServer != nil {
		s.ctlClientToServer.Close() // best effort
	}

	if s.ctlServerToClient != nil {
		s.ctlServerToClient.Close() // best effort
	}

	return s.mux.Close()
}

func config() *yamux.Config {
	cfg := yamux.DefaultConfig()

	//cfg.StreamOpenTimeout = 5 * time.Second // default=75s, yamux timeout will close the entire session, without reconnect we might as well terminate the program in that case

	cfg.LogOutput = nil
	cfg.Logger = log.New(io.Discard, "", log.LstdFlags) // discard all console logging in yamux

	return cfg
}
