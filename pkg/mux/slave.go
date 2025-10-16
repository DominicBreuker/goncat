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
// AcceptSessionContext creates a new slave session over the given connection
// while honoring the provided context for control-channel accepts. Tests and
// older call sites can continue to use AcceptSession which delegates to this
// function with context.Background().
func AcceptSessionContext(ctx context.Context, conn net.Conn) (*SlaveSession, error) {
	out := SlaveSession{
		sess: &Session{},
	}
	var err error

	out.sess.mux, err = yamux.Server(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Server(conn): %s", err)
	}

	out.sess.ctlClientToServer, err = out.AcceptNewChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlClientToServer: %s", err)
	}
	out.dec = gob.NewDecoder(out.sess.ctlClientToServer)

	out.sess.ctlServerToClient, err = out.AcceptNewChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel() for ctlServerToClient: %s", err)
	}
	out.enc = gob.NewEncoder(out.sess.ctlServerToClient)

	return &out, nil
}

// AcceptNewChannelContext accepts a new yamux stream using the provided context.
// It uses yamux's AcceptStreamWithContext when available to allow cancellation.
func (s *SlaveSession) AcceptNewChannelContext(ctx context.Context) (net.Conn, error) {
	// If the underlying yamux session supports AcceptStreamWithContext, use it.
	// The yamux Session's Accept() returns net.Conn; AcceptStreamWithContext
	// returns *yamux.Stream which implements net.Conn, but to avoid depending
	// on the exact type we rely on the exposed API from the yamux package.
	if s.sess == nil || s.sess.mux == nil {
		return nil, fmt.Errorf("no mux session")
	}

	// Try type assertion for the newer API. If it doesn't exist, fall back to Accept().
	type acceptWithCtx interface {
		AcceptStreamWithContext(context.Context) (*yamux.Stream, error)
	}

	if awc, ok := interface{}(s.sess.mux).(acceptWithCtx); ok {
		stream, err := awc.AcceptStreamWithContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("AcceptStreamWithContext(): %s", err)
		}
		if stream == nil {
			return nil, fmt.Errorf("AcceptStreamWithContext returned nil stream")
		}
		return stream, nil
	}

	// No context-aware accept available; use blocking Accept but respect ctx by
	// running accept in a goroutine and returning if ctx is done.
	type result struct {
		c   net.Conn
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		out, err := s.sess.mux.Accept()
		resCh <- result{out, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-resCh:
		if r.err != nil {
			return nil, fmt.Errorf("session.Accept(), ctl: %s", r.err)
		}
		return r.c, nil
	}
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
		return nil, fmt.Errorf("send(m): %s", err)
	}

	conn, err := s.AcceptNewChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("AcceptNewChannel(): %s", err)
	}

	return conn, nil
}

// ReceiveContext receives a message honoring the provided context's
// cancellation and deadline. The effective read deadline is the earlier of
// the caller's context deadline (if set) and now+ControlOpDeadline. If the
// caller provides a cancellable context without a deadline, we watch
// ctx.Done() and set the read deadline to now to interrupt blocking reads.
func (s *SlaveSession) ReceiveContext(ctx context.Context) (msg.Message, error) {
	var m msg.Message
	if s.sess != nil && s.sess.ctlClientToServer != nil {
		// compute absolute deadline
		dl := time.Now().Add(ControlOpDeadline)
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

	dl := time.Now().Add(ControlOpDeadline)
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
			deadline = time.Now().Add(ControlOpDeadline)
		}
		_ = s.sess.ctlServerToClient.SetWriteDeadline(deadline)
		defer s.sess.ctlServerToClient.SetWriteDeadline(time.Time{})
	}

	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %s", err)
	}

	return nil
}
