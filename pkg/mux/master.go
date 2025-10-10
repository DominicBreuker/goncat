package mux

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
)

// MasterSession represents the master side of a multiplexed connection.
// The master initiates the connection and sends commands to the slave.
// It uses gob encoding for message passing over dedicated control channels.
type MasterSession struct {
	sess *Session

	enc *gob.Encoder
	dec *gob.Decoder

	mu sync.Mutex
}

// Close closes the master session and its underlying multiplexed connection.
func (s *MasterSession) Close() error {
	return s.sess.Close()
}

// OpenSession creates a new master session over the given connection.
// It establishes a yamux client session and opens two control channels:
// one for client-to-server messages (with encoder) and one for server-to-client
// messages (with decoder).
func OpenSession(conn net.Conn) (*MasterSession, error) {
	out := MasterSession{
		sess: &Session{},
	}
	var err error

	out.sess.mux, err = yamux.Client(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Client(conn): %s", err)
	}

	out.sess.ctlClientToServer, err = out.openNewChannel()
	if err != nil {
		return nil, fmt.Errorf("out.OpenNewChannel() for ctlClientToServer: %s", err)
	}
	out.enc = gob.NewEncoder(out.sess.ctlClientToServer)

	out.sess.ctlServerToClient, err = out.openNewChannel()
	if err != nil {
		return nil, fmt.Errorf("out.OpenNewChannel() for ctlServerToClient: %s", err)
	}
	out.dec = gob.NewDecoder(out.sess.ctlServerToClient)

	return &out, nil
}

// SendAndGetOneChannel sends a message to the slave and opens a new channel
// for data transfer. This is used for operations that require one bidirectional
// data stream, such as port forwarding or SOCKS connections.
func (s *MasterSession) SendAndGetOneChannel(m msg.Message) (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Send(m); err != nil {
		return nil, fmt.Errorf("send(m): %s", err)
	}

	conn, err := s.openNewChannel()
	if err != nil {
		return nil, fmt.Errorf("openNewChannel(): %s", err)
	}

	return conn, nil
}

// SendAndGetTwoChannels sends a message to the slave and opens two new channels
// for data transfer. This is used for operations that require two unidirectional
// streams, such as separating stdin/stdout for command execution.
func (s *MasterSession) SendAndGetTwoChannels(m msg.Message) (net.Conn, net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Send(m); err != nil {
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

// GetOneChannel opens a new channel without sending a message first.
// This is used when the slave is expecting a channel to be opened by the master.
func (s *MasterSession) GetOneChannel() (net.Conn, error) {
	return s.openNewChannel()
}

// openNewChannel opens a new yamux stream over the multiplexed connection.
func (s *MasterSession) openNewChannel() (net.Conn, error) {
	out, err := s.sess.mux.Open()
	if err != nil {
		return nil, fmt.Errorf("session.Open(), ctl: %s", err)
	}

	return out, nil
}

// Send sends a message to the slave over the control channel using gob encoding.
func (s *MasterSession) Send(m msg.Message) error {
	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %s", err)
	}

	return nil
}

// Receive receives a message from the slave over the control channel using gob decoding.
func (s *MasterSession) Receive() (msg.Message, error) {
	var m msg.Message
	err := s.dec.Decode(&m)
	return m, err
}
