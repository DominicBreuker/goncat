package mux

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
)

// SlaveSession represents the slave side of a multiplexed connection.
// The slave accepts connections from the master and executes commands.
// It uses gob encoding for message passing over dedicated control channels.
type SlaveSession struct {
	sess *Session

	dec *gob.Decoder
	enc *gob.Encoder

	mu sync.Mutex
}

// Close closes the slave session and its underlying multiplexed connection.
func (s *SlaveSession) Close() error {
	return s.sess.Close()
}

// AcceptSession creates a new slave session over the given connection.
// It establishes a yamux server session and accepts two control channels:
// one for client-to-server messages (with decoder) and one for server-to-client
// messages (with encoder).
func AcceptSession(conn net.Conn) (*SlaveSession, error) {
	out := SlaveSession{
		sess: &Session{},
	}
	var err error

	out.sess.mux, err = yamux.Server(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Server(conn): %s", err)
	}

	out.sess.ctlClientToServer, err = out.AcceptNewChannel()
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlClientToServer: %s", err)
	}
	out.dec = gob.NewDecoder(out.sess.ctlClientToServer)

	out.sess.ctlServerToClient, err = out.AcceptNewChannel()
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlServerToClient: %s", err)
	}
	out.enc = gob.NewEncoder(out.sess.ctlServerToClient)

	return &out, nil
}

// SendAndGetOneChannel sends a message to the master and accepts a new channel
// for data transfer. This is used for operations that require one bidirectional
// data stream, such as port forwarding or SOCKS connections.
func (s *SlaveSession) SendAndGetOneChannel(m msg.Message) (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Send(m); err != nil {
		return nil, fmt.Errorf("send(m): %s", err)
	}

	conn, err := s.AcceptNewChannel()
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel(): %s", err)
	}

	return conn, nil
}

// GetOneChannel accepts a new channel without sending a message first.
// This is used when the master is opening a channel that the slave should accept.
func (s *SlaveSession) GetOneChannel() (net.Conn, error) {
	return s.AcceptNewChannel()
}

// AcceptNewChannel accepts a new yamux stream over the multiplexed connection.
func (s *SlaveSession) AcceptNewChannel() (net.Conn, error) {
	out, err := s.sess.mux.Accept()
	if err != nil {
		return nil, fmt.Errorf("session.Accept(), ctl: %s", err)
	}

	return out, nil
}

// Receive receives a message from the master over the control channel using gob decoding.
func (s *SlaveSession) Receive() (msg.Message, error) {
	var m msg.Message
	err := s.dec.Decode(&m)
	return m, err
}

// Send sends a message to the master over the control channel using gob encoding.
func (s *SlaveSession) Send(m msg.Message) error {
	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %s", err)
	}

	return nil
}
