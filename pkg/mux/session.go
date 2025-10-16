package mux

import (
	"io"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

// Session wraps a yamux.Session and two control streams.
// ctlClientToServer: master->slave (encode). ctlServerToClient: slave->master (decode).
type Session struct {
	mux *yamux.Session

	ctlClientToServer net.Conn
	ctlServerToClient net.Conn
}

// Close closes control streams then the yamux session (best-effort).
func (s *Session) Close() error {
	if s.ctlClientToServer != nil {
		s.ctlClientToServer.Close() // best effort
	}

	if s.ctlServerToClient != nil {
		s.ctlServerToClient.Close() // best effort
	}

	return s.mux.Close()
}

// config returns a yamux config with logging disabled.
// Keep default StreamOpenTimeout.
func config() *yamux.Config {
	cfg := yamux.DefaultConfig()

	// Keep default StreamOpenTimeout at 75s - yamux timeout closes the entire
	// session, and without reconnect we might as well terminate the program

	cfg.LogOutput = nil
	cfg.Logger = log.New(io.Discard, "", log.LstdFlags) // discard all console logging in yamux

	return cfg
}
