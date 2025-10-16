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

// OpenSessionContext creates a new master session over the given connection
// while honoring the provided context for opening the control channels.
// This mirrors OpenSession but allows callers to cancel the initial Open
// operations if needed. The legacy OpenSession delegates to this with a
// background context to preserve compatibility.
func OpenSessionContext(ctx context.Context, conn net.Conn) (*MasterSession, error) {
	out := MasterSession{
		sess: &Session{},
	}
	var err error

	out.sess.mux, err = yamux.Client(conn, config())
	if err != nil {
		return nil, fmt.Errorf("yamux.Client(conn): %s", err)
	}

	out.sess.ctlClientToServer, err = out.GetOneChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("out.GetOneChannelContext() for ctlClientToServer: %s", err)
	}
	out.enc = gob.NewEncoder(out.sess.ctlClientToServer)

	out.sess.ctlServerToClient, err = out.GetOneChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("out.GetOneChannelContext() for ctlServerToClient: %s", err)
	}
	out.dec = gob.NewDecoder(out.sess.ctlServerToClient)

	return &out, nil
}

// SendAndGetOneChannelContext sends a message to the slave and opens a new channel
// for data transfer, honoring the provided context for the Open operation.
func (s *MasterSession) SendAndGetOneChannelContext(ctx context.Context, m msg.Message) (net.Conn, error) {
	// Lock across send + opening the new channel to keep ordering atomic.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.sendLocked(m, time.Time{}); err != nil {
		return nil, fmt.Errorf("send(m): %s", err)
	}

	conn, err := s.GetOneChannelContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("openNewChannel(): %s", err)
	}

	return conn, nil
}

// SendAndGetTwoChannelsContext is the context-aware variant of SendAndGetTwoChannels.
// It sends a message and opens two channels, honoring ctx for the channel opens.
func (s *MasterSession) SendAndGetTwoChannelsContext(ctx context.Context, m msg.Message) (net.Conn, net.Conn, error) {
	// Lock across send + opening two channels to keep ordering atomic.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.sendLocked(m, time.Time{}); err != nil {
		return nil, nil, fmt.Errorf("send(m): %s", err)
	}

	conn1, err := s.GetOneChannelContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("GetOneChannelContext() for conn1: %s", err)
	}

	conn2, err := s.GetOneChannelContext(ctx)
	if err != nil {
		conn1.Close()
		return nil, nil, fmt.Errorf("GetOneChannelContext() for conn2: %s", err)
	}

	return conn1, conn2, nil
}

// GetOneChannelContext opens a new yamux stream using the provided context.
// It will use a context-aware Open if the underlying yamux session supports it,
// otherwise it falls back to running Open() in a goroutine and respects ctx.
func (s *MasterSession) GetOneChannelContext(ctx context.Context) (net.Conn, error) {
	if s.sess == nil || s.sess.mux == nil {
		return nil, fmt.Errorf("no mux session")
	}

	// Run blocking Open() in a goroutine and select on ctx.Done(). This is
	// the portable approach because the yamux package does not provide a
	// context-aware Open on the Session API. Using a goroutine + buffered
	// result channel lets us respect the caller's context while avoiding
	// goroutine leaks when the caller cancels early.
	type result struct {
		c   net.Conn
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		out, err := s.sess.mux.Open()
		resCh <- result{out, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-resCh:
		if r.err != nil {
			return nil, fmt.Errorf("session.Open(), ctl: %s", r.err)
		}
		return r.c, nil
	}
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
// SendContext sends a message with a context for cancellation.
func (s *MasterSession) SendContext(ctx context.Context, m msg.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// honour ctx cancellation by checking before write and using a deadline
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// compute absolute deadline: the earlier of caller's context deadline (if any)
	// and the default ControlOpDeadline from now.
	dl := time.Now().Add(ControlOpDeadline)
	if d, ok := ctx.Deadline(); ok && d.Before(dl) {
		dl = d
	}

	return s.sendLocked(m, dl)
}

// sendLocked sends a message assuming the caller already holds s.mu.
// It sets a write deadline on the control channel to bound blocking.
func (s *MasterSession) sendLocked(m msg.Message, deadline time.Time) error {
	if s.sess != nil && s.sess.ctlClientToServer != nil {
		// if caller passed zero time, fall back to ControlOpDeadline
		if deadline.IsZero() {
			deadline = time.Now().Add(ControlOpDeadline)
		}
		_ = s.sess.ctlClientToServer.SetWriteDeadline(deadline)
		defer s.sess.ctlClientToServer.SetWriteDeadline(time.Time{})
	}

	if err := s.enc.Encode(&m); err != nil {
		return fmt.Errorf("sending msg: %s", err)
	}

	return nil
}

// ReceiveContext receives a message honoring the provided context's
// cancellation and deadline. The effective read deadline is the earlier of
// the caller's context deadline (if set) and now+ControlOpDeadline. If the
// caller provides a cancellable context without a deadline, we watch
// ctx.Done() and set the read deadline to now to interrupt blocking reads.
func (s *MasterSession) ReceiveContext(ctx context.Context) (msg.Message, error) {
	var m msg.Message
	if s.sess != nil && s.sess.ctlServerToClient != nil {
		// compute absolute deadline
		dl := time.Now().Add(ControlOpDeadline)
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
