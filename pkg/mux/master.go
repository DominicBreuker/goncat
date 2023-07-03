package mux

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"net"

	"github.com/hashicorp/yamux"
)

// MasterSession ...
type MasterSession struct {
	sess *session

	enc *gob.Encoder
}

// Close ...
func (s *MasterSession) Close() error {
	return s.sess.Close()
}

// OpenSession ...
func OpenSession(conn net.Conn) (*MasterSession, error) {
	out := MasterSession{
		sess: &session{},
	}
	var err error

	out.sess.mux, err = yamux.Client(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Client(conn): %s", err)
	}

	out.sess.connCtl, err = out.OpenNewChannel()
	if err != nil {
		return nil, fmt.Errorf("out.OpenNewChannel(): %s", err)
	}

	out.enc = gob.NewEncoder(out.sess.connCtl)

	return &out, nil
}

// OpenNewChannel ...
func (s *MasterSession) OpenNewChannel() (net.Conn, error) {
	out, err := s.sess.mux.Open()
	if err != nil {
		return nil, fmt.Errorf("session.Open(), ctl: %s", err)
	}

	return out, nil
}

// Send ...
func (s *MasterSession) Send(m msg.Message) error {
	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %s", err)
	}

	return nil
}
