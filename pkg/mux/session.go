package mux

import (
	"io/ioutil"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

// Session ...
type session struct {
	mux *yamux.Session

	connCtl net.Conn
}

// Close ...
func (s *session) Close() error {
	if s.connCtl != nil {
		s.connCtl.Close() // best effort
	}

	return s.mux.Close()
}

func config() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = nil
	cfg.Logger = log.New(ioutil.Discard, "", log.LstdFlags) // discard all console logging in yamux
	return cfg
}
