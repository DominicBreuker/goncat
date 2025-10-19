// Package master provides the master-side handler for managing multiplexed
// connections. The master controls the connection, initiating port forwarding,
// SOCKS proxies, and foreground tasks that the slave executes.
package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
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
type Master struct {
	ctx  context.Context
	cfg  *config.Shared
	mCfg *config.Master

	remoteAddr string
	remoteID   string

	sess *mux.MasterSession
}

// New creates a new Master handler over the given connection.
// It opens a multiplexed session for managing multiple concurrent operations.
func New(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (*Master, error) {
	sess, err := mux.OpenSessionContext(context.Background(), conn, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("mux.OpenSession(conn): %s", err)
	}

	remoteAddr := conn.RemoteAddr().String()

	return &Master{
		ctx:        ctx,
		cfg:        cfg,
		mCfg:       mCfg,
		remoteAddr: remoteAddr,
		sess:       sess,
	}, nil
}

// Close closes the master's multiplexed session and all associated resources.
func (mst *Master) Close() error {
	return mst.sess.Close()
}

// Handle starts all configured master operations (port forwarding, SOCKS, foreground task)
// and processes incoming messages from the slave. It blocks until all operations complete.
func (mst *Master) Handle() error {
	// let user know about connection status
	defer func() {
		if mst.remoteID != "" {
			log.InfoMsg("Session with %s closed (%s)\n", mst.remoteAddr, mst.remoteID)
		}
	}()

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(mst.ctx)

	for _, lpf := range mst.mCfg.LocalPortForwarding {
		mst.startLocalPortFwdJob(ctx, &wg, lpf)
	}
	for _, rpf := range mst.mCfg.RemotePortForwarding {
		mst.startRemotePortFwdJob(ctx, &wg, rpf)
	}

	if mst.mCfg.IsSocksEnabled() {
		if err := mst.startSocksProxyJob(ctx, &wg); err != nil {
			log.ErrorMsg("Starting SOCKS proxy: %s", err)
		}
	}

	mst.startForegroundJob(ctx, &wg, cancel) // foreground job must cancel when it terminates

	if err := mst.sess.SendContext(ctx, msg.Hello{
		ID: mst.cfg.ID,
	}); err != nil {
		log.ErrorMsg("sending hello to slave: %s\n", err)
	}

	go func() {
		for {
			m, err := mst.sess.ReceiveContext(ctx)
			if err != nil {
				if err == io.EOF {
					return
				}
				if ctx.Err() != nil {
					return // cancelled
				}

				// Ignore expected timeouts/deadlines (frequent when polling).
				if err == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}

				log.ErrorMsg("Receiving next command: %s\n", err)
				continue
			}

			switch message := m.(type) {
			case msg.Hello:
				// let user know about connection status
				mst.remoteID = message.ID
				log.InfoMsg("Session with %s established (%s)\n", mst.remoteAddr, mst.remoteID)
			case msg.Connect:
				// validate messages from slave to ensure we only forward to destintions specified in master configuration
				if !mst.mCfg.IsAllowedRemotePortForwardingDestination(message.RemoteHost, message.RemotePort) {
					log.ErrorMsg("Remote port forwarding: slave requested unexpected destination: %s:%d\n", message.RemoteHost, message.RemotePort)
					continue
				}
				mst.handleConnectAsync(ctx, message)

			default:
				log.ErrorMsg("Received unsupported message type '%s', this is a bug\n", m.MsgType())
			}
		}
	}()

	wg.Wait()

	return nil
}
