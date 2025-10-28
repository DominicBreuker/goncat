// Package master provides the master-side handler for managing multiplexed
// connections. The master controls the connection, initiating port forwarding,
// SOCKS proxies, and foreground tasks that the slave executes.
package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux"
	"dominicbreuker/goncat/pkg/mux/msg"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// Master manages the master side of a multiplexed connection, coordinating
// port forwarding, SOCKS proxies, and command execution on the slave.
// master is the package-local state for a master handler.
type master struct {
	ctx  context.Context
	cfg  *config.Shared
	mCfg *config.Master

	remoteAddr string
	remoteID   string

	sess *mux.MasterSession
}

// Handle creates a master handler over the given connection and runs it until completion.
func Handle(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) error {
	mst := &master{
		ctx:        ctx,
		cfg:        cfg,
		mCfg:       mCfg,
		remoteAddr: conn.RemoteAddr().String(),
		sess:       nil,
	}
	var err error

	cfg.Logger.VerboseMsg("Master handler starting for connection from %s", conn.RemoteAddr())

	// let user know about connection status
	defer func() {
		if mst.remoteID != "" {
			cfg.Logger.InfoMsg("Session with %s closed (%s)\n", mst.remoteAddr, mst.remoteID)
			cfg.Logger.VerboseMsg("Closing session with %s (%s)", mst.remoteAddr, mst.remoteID)
		}
	}()

	cfg.Logger.VerboseMsg("Opening yamux session with %s", mst.remoteAddr)
	mst.sess, err = mux.OpenSessionContext(ctx, conn, cfg.Timeout)
	if err != nil {
		cfg.Logger.VerboseMsg("Failed to open yamux session: %v", err)
		return fmt.Errorf("mux.OpenSession(conn): %s", err)
	}
	defer func() { _ = mst.sess.Close() }()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(mst.ctx)
	defer cancel()

	// 1) Perform handshake: send Hello and wait for Hello
	cfg.Logger.VerboseMsg("Sending Hello message to slave")
	if err := mst.sess.SendContext(ctx, msg.Hello{ID: mst.cfg.ID}); err != nil {
		// Treat handshake send failure as terminal
		cfg.Logger.VerboseMsg("Failed to send Hello message: %v", err)
		return fmt.Errorf("sending hello to slave: %w", err)
	}

	// Give the peer a bounded time to respond to Hello
	helloCtx, helloCancel := context.WithTimeout(ctx, mst.cfg.Timeout)
	defer helloCancel()

	cfg.Logger.VerboseMsg("Waiting for Hello response from slave")
	helloSeen := false
	for !helloSeen {
		m, err := mst.sess.ReceiveContext(helloCtx)
		if err != nil {
			if err == io.EOF {
				cfg.Logger.VerboseMsg("Handshake failed: peer closed connection")
				return fmt.Errorf("handshake: peer closed")
			}
			if helloCtx.Err() != nil || ctx.Err() != nil {
				cfg.Logger.VerboseMsg("Handshake timeout: %v", helloCtx.Err())
				return fmt.Errorf("handshake: %w", helloCtx.Err())
			}
			// Common transient/timeout errors during early handshake; keep waiting within helloCtx
			if err == context.DeadlineExceeded {
				return fmt.Errorf("handshake timeout waiting for hello")
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				// keep waiting until helloCtx expires
				continue
			}
			// Any other error: still fail handshake
			return fmt.Errorf("handshake receive: %w", err)
		}

		switch message := m.(type) {
		case msg.Hello:
			mst.remoteID = message.ID
			cfg.Logger.InfoMsg("Session with %s established (%s)\n", mst.remoteAddr, mst.remoteID)
			cfg.Logger.VerboseMsg("Received Hello from slave %s (ID: %s)", mst.remoteAddr, mst.remoteID)
			helloSeen = true
		default:
			// Ignore any other messages until hello is completed.
			// (In practice slave won’t send others before handshake.)
		}
	}

	// 2) Start jobs AFTER handshake completes
	for _, lpf := range mst.mCfg.LocalPortForwarding {
		mst.startLocalPortFwdJob(ctx, &wg, lpf)
	}
	for _, rpf := range mst.mCfg.RemotePortForwarding {
		mst.startRemotePortFwdJob(ctx, &wg, rpf)
	}
	if mst.mCfg.IsSocksEnabled() {
		if err := mst.startSocksProxyJob(ctx, &wg); err != nil {
			cfg.Logger.ErrorMsg("Starting SOCKS proxy: %s", err)
		}
	}

	// Foreground cancels the whole session when it finishes
	mst.startForegroundJob(ctx, &wg, cancel)

	// 3) Background receive loop for post-handshake messages
	go func() {
		for {
			m, err := mst.sess.ReceiveContext(ctx)
			if err != nil {
				if err == io.EOF || ctx.Err() != nil {
					return
				}
				// Ignore polling timeouts/deadlines
				if err == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				cfg.Logger.ErrorMsg("Receiving next command: %s\n", err)
				continue
			}

			switch message := m.(type) {
			case msg.Connect:
				// Validate RPF destination
				if !mst.mCfg.IsAllowedRemotePortForwardingDestination(message.RemoteHost, message.RemotePort) {
					cfg.Logger.ErrorMsg("Remote port forwarding: slave requested unexpected destination: %s:%d\n",
						message.RemoteHost, message.RemotePort)
					continue
				}
				mst.handleConnectAsync(ctx, message)

			case msg.Hello:
				// Duplicate hello after handshake: harmless; ignore.
			default:
				cfg.Logger.ErrorMsg("Received unsupported message type '%s', this is a bug\n", m.MsgType())
			}
		}
	}()

	// 4) Wait for all jobs to finish (foreground will call cancel())
	wg.Wait()
	return nil
}
