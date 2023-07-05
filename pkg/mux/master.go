package mux

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
)

// MasterSession ...
type MasterSession struct {
	sess *session

	enc *gob.Encoder
	mu  sync.Mutex
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

	out.sess.connCtl, err = out.openNewChannel()
	if err != nil {
		return nil, fmt.Errorf("out.OpenNewChannel(): %s", err)
	}

	out.enc = gob.NewEncoder(out.sess.connCtl)

	return &out, nil
}

// SendAndOpenOneChannel ...
func (s *MasterSession) SendAndOpenOneChannel(m msg.Message) (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.send(m); err != nil {
		return nil, fmt.Errorf("send(m): %s", err)
	}

	conn, err := s.openNewChannel()
	if err != nil {
		return nil, fmt.Errorf("openNewChannel(): %s", err)
	}

	return conn, nil
}

// SendAndOpenTwoChannels ...
func (s *MasterSession) SendAndOpenTwoChannels(m msg.Message) (net.Conn, net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.send(m); err != nil {
		return nil, nil, fmt.Errorf("send(m): %s", err)
	}

	conn1, err := s.openNewChannel()
	if err != nil {
		return nil, nil, fmt.Errorf("openNewChannel() for conn1: %s", err)
	}

	conn2, err := s.openNewChannel()
	if err != nil {
		conn1.Close()
		return nil, nil, fmt.Errorf("openNewChannel() for conn2: %s", err)
	}

	return conn1, conn2, nil
}

// OpenNewChannel ...
func (s *MasterSession) openNewChannel() (net.Conn, error) {
	out, err := s.sess.mux.Open()
	if err != nil {
		return nil, fmt.Errorf("session.Open(), ctl: %s", err)
	}

	return out, nil
}

// Send ...
func (s *MasterSession) send(m msg.Message) error {
	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %s", err)
	}

	return nil
}
