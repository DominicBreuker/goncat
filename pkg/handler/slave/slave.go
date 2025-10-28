// Package slave provides the slave-side handler for responding to commands
// from the master over a multiplexed connection. The slave executes commands,
// handles port forwarding requests, and manages SOCKS proxy connections.
package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux"
	"dominicbreuker/goncat/pkg/mux/msg"
	"errors"
	"fmt"
	"io"
	"net"
)

// slave is the package-local state for a slave handler.
type slave struct {
	ctx context.Context
	cfg *config.Shared

	remoteAddr string
	remoteID   string

	sess *mux.SlaveSession
}

// Handle creates a slave handler over the given connection and runs it until completion.
func Handle(ctx context.Context, cfg *config.Shared, conn net.Conn) error {
	cfg.Logger.VerboseMsg("Slave handler starting for connection from %s", conn.RemoteAddr())
	slv := &slave{
		ctx:        ctx,
		cfg:        cfg,
		remoteAddr: conn.RemoteAddr().String(),
		sess:       nil,
	}

	// let user know about connection status
	defer func() {
		if slv.remoteID != "" {
			slv.cfg.Logger.InfoMsg("Session with %s closed (%s)\n", slv.remoteAddr, slv.remoteID)
			slv.cfg.Logger.VerboseMsg("Closing session with %s (%s)", slv.remoteAddr, slv.remoteID)
		}
	}()

	var err error
	cfg.Logger.VerboseMsg("Accepting yamux session from %s", conn.RemoteAddr())
	slv.sess, err = mux.AcceptSessionContext(ctx, conn, cfg.Timeout)
	if err != nil {
		cfg.Logger.VerboseMsg("Failed to accept yamux session: %v", err)
		return fmt.Errorf("mux.AcceptSession(conn): %s", err)
	}
	defer func() { _ = slv.sess.Close() }()

	return slv.run()
}

// Close closes the slave's multiplexed session and all associated resources.
func (slv *slave) Close() error {
	if slv.sess == nil {
		return nil
	}
	return slv.sess.Close()
}

// run contains the former Slave.Handle implementation and runs on an already-initialized slave.
func (slv *slave) run() error {
	ctx, cancel := context.WithCancel(slv.ctx)
	defer cancel()

	// 1) Send Hello
	slv.cfg.Logger.VerboseMsg("Sending Hello message to master")
	if err := slv.sess.SendContext(ctx, msg.Hello{ID: slv.cfg.ID}); err != nil {
		slv.cfg.Logger.VerboseMsg("Failed to send Hello message: %v", err)
		// treat as terminal; session likely unusable
		return fmt.Errorf("sending hello to master: %w", err)
	}

	// 2) Handshake barrier: wait for master's Hello within timeout
	helloCtx, helloCancel := context.WithTimeout(ctx, slv.cfg.Timeout)
	slv.cfg.Logger.VerboseMsg("Waiting for Hello from master")
	defer helloCancel()

	for {
		m, err := slv.sess.ReceiveContext(helloCtx)
		if err != nil {
			if err == io.EOF {
				slv.cfg.Logger.VerboseMsg("Handshake failed: peer closed connection")
				return fmt.Errorf("handshake: peer closed")
			}
			if helloCtx.Err() != nil || ctx.Err() != nil {
				// timed out or cancelled while waiting for hello
				return fmt.Errorf("handshake: %w", helloCtx.Err())
			}
			// Ignore polling-related timeouts; keep waiting within helloCtx
			if err == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			// Any other error: fail handshake
			return fmt.Errorf("handshake receive: %w", err)
		}

		if h, ok := m.(msg.Hello); ok {
			slv.cfg.Logger.VerboseMsg("Received Hello from master %s (ID: %s)", slv.remoteAddr, h.ID)
			slv.remoteID = h.ID
			slv.cfg.Logger.InfoMsg("Session with %s established (%s)\n", slv.remoteAddr, slv.remoteID)
			break
		}
		// Ignore any other message types until Hello is seen (shouldn’t happen).
	}

	// 3) Main loop: process commands
	for {
		m, err := slv.sess.ReceiveContext(ctx)
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return nil
			}
			// Ignore deadline/timeout errors caused by polling checks.
			if err == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}

			slv.cfg.Logger.ErrorMsg("Receiving next command: %s\n", err)
			continue
		}

		switch message := m.(type) {
		case msg.Hello:
			// Duplicate Hello after barrier: harmless; ignore.
		case msg.Foreground:
			slv.handleForegroundAsync(ctx, message)
		case msg.Connect:
			slv.handleConnectAsync(ctx, message)
		case msg.PortFwd:
			slv.handlePortFwdAsync(ctx, message)
		case msg.SocksConnect:
			slv.handleSocksConnectAsync(ctx, message)
		case msg.SocksAssociate:
			slv.handleSocksAsociateAsync(ctx, message)
		default:
			return fmt.Errorf("unsupported message type '%s', this is a bug", m.MsgType())
		}
	}
}
