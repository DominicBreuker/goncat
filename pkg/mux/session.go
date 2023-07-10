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

	ctlMasterToSlave net.Conn
	ctlSlaveToMaster net.Conn
}

// Close ...
func (s *session) Close() error {
	if s.ctlMasterToSlave != nil {
		s.ctlMasterToSlave.Close() // best effort
	}

	if s.ctlSlaveToMaster != nil {
		s.ctlSlaveToMaster.Close() // best effort
	}

	return s.mux.Close()
}

func config() *yamux.Config {
	cfg := yamux.DefaultConfig()

	//cfg.StreamOpenTimeout = 5 * time.Second // default=75s, yamux timeout will close the entire session, without reconnect we might as well terminate the program in that case

	cfg.LogOutput = nil
	cfg.Logger = log.New(ioutil.Discard, "", log.LstdFlags) // discard all console logging in yamux

	return cfg
}
