package mux

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"net"

	"github.com/hashicorp/yamux"
)

// SlaveSession ...
type SlaveSession struct {
	sess *session

	dec *gob.Decoder
}

// Close ...
func (s SlaveSession) Close() error {
	return s.sess.Close()
}

// AcceptSession ...
func AcceptSession(conn net.Conn) (*SlaveSession, error) {
	out := SlaveSession{
		sess: &session{},
	}
	var err error

	out.sess.mux, err = yamux.Server(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Server(conn): %s", err)
	}

	out.sess.connCtl, err = out.AcceptNewChannel()
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel(): %s", err)
	}

	out.dec = gob.NewDecoder(out.sess.connCtl)

	return &out, nil
}

// AcceptNewChannel ...
func (s *SlaveSession) AcceptNewChannel() (net.Conn, error) {
	out, err := s.sess.mux.Accept()
	if err != nil {
		return nil, fmt.Errorf("session.Accept(), ctl: %s", err)
	}

	return out, nil
}

// Receive ...
func (s *SlaveSession) Receive() (msg.Message, error) {
	var m msg.Message
	err := s.dec.Decode(&m)
	return m, err
}
