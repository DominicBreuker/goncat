package mux

import (
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

// SlaveSession represents the slave side of a multiplexed connection.
// The slave accepts connections from the master and executes commands.
// It uses gob encoding for message passing over dedicated control channels.
type SlaveSession struct {
	sess *Session

	dec *gob.Decoder
	enc *gob.Encoder

	timeout time.Duration

	mu sync.Mutex
}

// Close closes the slave session and its underlying multiplexed connection.
func (s *SlaveSession) Close() error {
	return s.sess.Close()
}

// AcceptSessionContext creates a new slave session over the given connection
// while honoring the provided context for control-channel accepts. timeout specifies
// the deadline for control operations.
func AcceptSessionContext(ctx context.Context, conn net.Conn, timeout time.Duration) (*SlaveSession, error) {
	out := SlaveSession{
		sess:    &Session{},
		timeout: timeout,
	}
	var err error

	out.sess.mux, err = yamux.Server(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Server(conn): %w", err)
	}

	out.sess.ctlClientToServer, err = out.AcceptNewChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlClientToServer: %w", err)
	}
	out.dec = gob.NewDecoder(out.sess.ctlClientToServer)

	out.sess.ctlServerToClient, err = out.AcceptNewChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlServerToClient: %w", err)
	}
	out.enc = gob.NewEncoder(out.sess.ctlServerToClient)

	return &out, nil
}

// AcceptNewChannelContext accepts a new yamux stream using the provided context.
// It uses yamux's AcceptStreamWithContext when available to allow cancellation.
func (s *SlaveSession) AcceptNewChannelContext(ctx context.Context) (net.Conn, error) {
	if s.sess == nil || s.sess.mux == nil {
		return nil, fmt.Errorf("no mux session")
	}

	// enforce timeout while accepting control channels
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(s.timeout))
	defer cancel()

	stream, err := s.sess.mux.AcceptStreamWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("AcceptStreamWithContext(): %w", err)
	}
	if stream == nil {
		return nil, fmt.Errorf("AcceptStreamWithContext returned nil stream")
	}
	return stream, nil
}

// GetOneChannelContext accepts a new channel using the provided context.
func (s *SlaveSession) GetOneChannelContext(ctx context.Context) (net.Conn, error) {
	return s.AcceptNewChannelContext(ctx)
}

// SendAndGetOneChannelContext sends a message to the master and accepts a new channel
// for data transfer, using the provided context for cancellation of the accept.
func (s *SlaveSession) SendAndGetOneChannelContext(ctx context.Context, m msg.Message) (net.Conn, error) {
	// Lock across send + accepting the new channel to keep ordering atomic
	// with respect to the master.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.sendLocked(m, time.Time{}); err != nil {
		return nil, fmt.Errorf("send(m): %w", err)
	}

	conn, err := s.AcceptNewChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel(): %w", err)
	}

	return conn, nil
}

// ReceiveContext receives a message honoring the provided context's
// cancellation and deadline. The effective read deadline is the earlier of
// the caller's context deadline (if set) and the configured timeout. If the
// caller provides a cancellable context without a deadline, we watch
// ctx.Done() and set the read deadline to now to interrupt blocking reads.
func (s *SlaveSession) ReceiveContext(ctx context.Context) (msg.Message, error) {
	if s.sess != nil && s.sess.ctlClientToServer != nil {
		// compute absolute deadline
		dl := time.Now().Add(s.timeout)
		if d, ok := ctx.Deadline(); ok && d.Before(dl) {
			dl = d
		}

		_ = s.sess.ctlClientToServer.SetReadDeadline(dl)
		defer s.sess.ctlClientToServer.SetReadDeadline(time.Time{})

		if _, ok := ctx.Deadline(); !ok {
			done := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					_ = s.sess.ctlClientToServer.SetReadDeadline(time.Now())
				case <-done:
				}
			}()
			defer close(done)
		}
	}

	var m msg.Message
	err := s.dec.Decode(&m)
	return m, err
}

// Send sends a message to the master over the control channel using gob encoding.
// SendContext sends a message with a context for cancellation.
func (s *SlaveSession) SendContext(ctx context.Context, m msg.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	dl := time.Now().Add(s.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(dl) {
		dl = d
	}

	return s.sendLocked(m, dl)
}

// sendLocked sends a message assuming the caller already holds s.mu.
// It sets a write deadline on the control channel to bound blocking.
func (s *SlaveSession) sendLocked(m msg.Message, deadline time.Time) error {
	if s.sess != nil && s.sess.ctlServerToClient != nil {
		if deadline.IsZero() {
			deadline = time.Now().Add(s.timeout)
		}
		_ = s.sess.ctlServerToClient.SetWriteDeadline(deadline)
		defer s.sess.ctlServerToClient.SetWriteDeadline(time.Time{})
	}

	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %w", err)
	}

	return nil
}
