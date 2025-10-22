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

// MasterSession is the master-side mux wrapper. It encodes to ctlClientToServer
// and decodes from ctlServerToClient using gob.
type MasterSession struct {
	sess *Session

	enc *gob.Encoder
	dec *gob.Decoder

	timeout time.Duration

	mu sync.Mutex
}

// Close closes the session.
func (s *MasterSession) Close() error {
	if s.sess == nil {
		return nil
	}
	return s.sess.Close()
}

// OpenSessionContext creates a master session and opens the two control streams.
// ctx cancels control stream opens. timeout specifies the deadline for control operations.
func OpenSessionContext(ctx context.Context, conn net.Conn, timeout time.Duration) (*MasterSession, error) {
	out := MasterSession{
		sess:    &Session{},
		timeout: timeout,
	}
	var err error

	out.sess.mux, err = yamux.Client(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Client(conn): %w", err)
	}

	out.sess.ctlClientToServer, err = out.GetOneChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("out.GetOneChannelContext() for ctlClientToServer: %w", err)
	}
	out.enc = gob.NewEncoder(out.sess.ctlClientToServer)

	out.sess.ctlServerToClient, err = out.GetOneChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("out.GetOneChannelContext() for ctlServerToClient: %w", err)
	}
	out.dec = gob.NewDecoder(out.sess.ctlServerToClient)

	return &out, nil
}

// SendAndGetOneChannelContext sends m and opens one yamux stream. The send
// and stream-open are ordered by holding s.mu.
func (s *MasterSession) SendAndGetOneChannelContext(ctx context.Context, m msg.Message) (net.Conn, error) {
	// Lock across send + opening the new channel to keep ordering atomic.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.sendLocked(m, time.Time{}); err != nil {
		return nil, fmt.Errorf("send(m): %w", err)
	}

	conn, err := s.GetOneChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("openNewChannel(): %w", err)
	}

	return conn, nil
}

// SendAndGetTwoChannelsContext sends m and opens two yamux streams atomically.
func (s *MasterSession) SendAndGetTwoChannelsContext(ctx context.Context, m msg.Message) (net.Conn, net.Conn, error) {
	// Lock across send + opening two channels to keep ordering atomic.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.sendLocked(m, time.Time{}); err != nil {
		return nil, nil, fmt.Errorf("send(m): %w", err)
	}

	conn1, err := s.GetOneChannelContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("GetOneChannelContext() for conn1: %w", err)
	}

	conn2, err := s.GetOneChannelContext(ctx)
	if err != nil {
		conn1.Close()
		return nil, nil, fmt.Errorf("GetOneChannelContext() for conn2: %w", err)
	}

	return conn1, conn2, nil
}

// GetOneChannelContext opens a yamux stream with ctx cancellation and a default timeout.
func (s *MasterSession) GetOneChannelContext(ctx context.Context) (net.Conn, error) {
	if s.sess == nil || s.sess.mux == nil {
		return nil, fmt.Errorf("no mux session")
	}

	// If the caller provided no deadline, derive one from s.timeout.
	if _, has := ctx.Deadline(); !has && s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	type result struct {
		c   net.Conn
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		out, err := s.sess.mux.Open()
		// Buffered channel means this goroutine never blocks even if the caller returns early.
		resCh <- result{out, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-resCh:
		if r.err != nil {
			return nil, fmt.Errorf("session.Open(), ctl: %w", r.err)
		}
		return r.c, nil
	}
}

// SendContext encodes m on the control channel. ctx cancels or shortens the op.
func (s *MasterSession) SendContext(ctx context.Context, m msg.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// honour ctx cancellation by checking before write and using a deadline
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// compute absolute deadline: the earlier of caller's context deadline (if any)
	// and the default timeout from configuration.
	dl := time.Now().Add(s.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(dl) {
		dl = d
	}

	return s.sendLocked(m, dl)
}

// sendLocked encodes m while s.mu is held and sets a write deadline.
func (s *MasterSession) sendLocked(m msg.Message, deadline time.Time) error {
	if s.sess != nil && s.sess.ctlClientToServer != nil {
		// if caller passed zero time, fall back to timeout from configuration
		if deadline.IsZero() {
			deadline = time.Now().Add(s.timeout)
		}
		_ = s.sess.ctlClientToServer.SetWriteDeadline(deadline)
		defer s.sess.ctlClientToServer.SetWriteDeadline(time.Time{})
	}

	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %w", err)
	}

	return nil
}

// ReceiveContext decodes a message from the control stream, honoring ctx.
// If ctx has no deadline we watch ctx.Done() and set an immediate read
// deadline to interrupt Decode.
func (s *MasterSession) ReceiveContext(ctx context.Context) (msg.Message, error) {
	var m msg.Message
	if s.sess != nil && s.sess.ctlServerToClient != nil {
		// compute absolute deadline
		dl := time.Now().Add(s.timeout)
		if d, ok := ctx.Deadline(); ok && d.Before(dl) {
			dl = d
		}

		_ = s.sess.ctlServerToClient.SetReadDeadline(dl)
		defer s.sess.ctlServerToClient.SetReadDeadline(time.Time{})

		// if caller didn't provide a deadline, allow ctx cancellation to
		// interrupt the blocking decode by setting an immediate read deadline
		// when ctx is cancelled. Use a done channel to avoid goroutine leaks.
		if _, ok := ctx.Deadline(); !ok {
			done := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					_ = s.sess.ctlServerToClient.SetReadDeadline(time.Now())
				case <-done:
				}
			}()
			defer close(done)
		}
	}

	err := s.dec.Decode(&m)
	return m, err
}
