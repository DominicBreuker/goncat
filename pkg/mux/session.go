// Package mux provides multiplexing functionality for goncat connections.
// It wraps yamux to enable multiple logical streams over a single underlying
// connection, with bidirectional control channels for message passing between
// master and slave.
package mux

import (
	"io"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

// Session represents a multiplexed connection with control channels for
// bidirectional message passing. It wraps a yamux session and maintains
// dedicated control channels for client-to-server and server-to-client
// communication.
type Session struct {
	mux *yamux.Session

	ctlClientToServer net.Conn
	ctlServerToClient net.Conn
}

// Close closes the session and its control channels. It attempts to close
// both control channels (best effort) before closing the underlying yamux
// session.
func (s *Session) Close() error {
	if s.ctlClientToServer != nil {
		s.ctlClientToServer.Close() // best effort
	}

	if s.ctlServerToClient != nil {
		s.ctlServerToClient.Close() // best effort
	}

	return s.mux.Close()
}

// config returns a yamux configuration with logging disabled.
// The default stream open timeout is kept at 75 seconds, as yamux timeouts
// will close the entire session and without reconnect support, the program
// would terminate anyway.
func config() *yamux.Config {
	cfg := yamux.DefaultConfig()

	// Keep default StreamOpenTimeout at 75s - yamux timeout closes the entire
	// session, and without reconnect we might as well terminate the program

	cfg.LogOutput = nil
	cfg.Logger = log.New(io.Discard, "", log.LstdFlags) // discard all console logging in yamux

	return cfg
}
