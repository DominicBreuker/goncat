package mux

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
)

// SlaveSession ...
type SlaveSession struct {
	sess *session

	dec *gob.Decoder
	enc *gob.Encoder

	mu sync.Mutex
}

// Close ...
func (s *SlaveSession) Close() error {
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

	out.sess.ctlMasterToSlave, err = out.AcceptNewChannel()
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlMasterToSlave: %s", err)
	}
	out.dec = gob.NewDecoder(out.sess.ctlMasterToSlave)

	out.sess.ctlSlaveToMaster, err = out.AcceptNewChannel()
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlSlaveToMaster: %s", err)
	}
	out.enc = gob.NewEncoder(out.sess.ctlSlaveToMaster)

	return &out, nil
}

// SendAndGetOneChannel ...
func (s *SlaveSession) SendAndGetOneChannel(m msg.Message) (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.send(m); err != nil {
		return nil, fmt.Errorf("send(m): %s", err)
	}

	conn, err := s.AcceptNewChannel()
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel(): %s", err)
	}

	return conn, nil
}

// GetOneChannel ...
func (s *SlaveSession) GetOneChannel() (net.Conn, error) {
	return s.AcceptNewChannel()
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

// Send ...
func (s *SlaveSession) send(m msg.Message) error {
	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %s", err)
	}

	return nil
}
