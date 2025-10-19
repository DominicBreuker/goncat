// Package slave provides the slave-side handler for responding to commands
// from the master over a multiplexed connection. The slave executes commands,
// handles port forwarding requests, and manages SOCKS proxy connections.
package slave

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
)

// Slave manages the slave side of a multiplexed connection, responding to
// commands from the master for execution, port forwarding, and proxying.
type Slave struct {
	ctx context.Context
	cfg *config.Shared

	remoteAddr string
	remoteID   string

	sess *mux.SlaveSession
}

// New creates a new Slave handler over the given connection.
// It accepts a multiplexed session for handling commands from the master.
func New(ctx context.Context, cfg *config.Shared, conn net.Conn) (*Slave, error) {
	sess, err := mux.AcceptSessionContext(ctx, conn, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("mux.AcceptSession(conn): %s", err)
	}

	remoteAddr := conn.RemoteAddr().String()

	return &Slave{
		ctx:        ctx,
		cfg:        cfg,
		remoteAddr: remoteAddr,
		sess:       sess,
	}, nil
}

// Close closes the slave's multiplexed session and all associated resources.
func (slv *Slave) Close() error {
	return slv.sess.Close()
}

// Handle processes incoming messages from the master and dispatches them to
// the appropriate handlers. It blocks until the connection is closed or an error occurs.
func (slv *Slave) Handle() error {
	// let user know about connection status
	defer func() {
		if slv.remoteID != "" {
			log.InfoMsg("Session with %s closed (%s)\n", slv.remoteAddr, slv.remoteID)
		}
	}()

	ctx, cancel := context.WithCancel(slv.ctx)
	defer cancel()

	if err := slv.sess.SendContext(ctx, msg.Hello{
		ID: slv.cfg.ID,
	}); err != nil {
		log.ErrorMsg("sending hello to master: %s\n", err)
	}

	for {
		m, err := slv.sess.ReceiveContext(slv.ctx)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			// Ignore deadline/timeout errors caused by context/deadline checks.
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
			slv.remoteID = message.ID
			log.InfoMsg("Session with %s established (%s)\n", slv.remoteAddr, slv.remoteID)
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
