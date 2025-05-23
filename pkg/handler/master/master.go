package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux"
	"dominicbreuker/goncat/pkg/mux/msg"
	"fmt"
	"io"
	"net"
	"sync"
)

// Master ...
type Master struct {
	ctx  context.Context
	cfg  *config.Shared
	mCfg *config.Master

	sess *mux.MasterSession
}

// New ...
func New(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) (*Master, error) {
	sess, err := mux.OpenSession(conn)
	if err != nil {
		return nil, fmt.Errorf("mux.OpenSession(conn): %s", err)
	}

	return &Master{
		ctx:  ctx,
		cfg:  cfg,
		mCfg: mCfg,
		sess: sess,
	}, nil
}

// Close ...
func (mst *Master) Close() error {
	return mst.sess.Close()
}

// Handle ...
func (mst *Master) Handle() error {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(mst.ctx)

	for _, lpf := range mst.mCfg.LocalPortForwarding {
		mst.startLocalPortFwdJobJob(ctx, &wg, lpf)
	}
	for _, rpf := range mst.mCfg.RemotePortForwarding {
		mst.startRemotePortFwdJobJob(ctx, &wg, rpf)
	}

	if mst.mCfg.IsSocksEnabled() {
		if err := mst.startSocksProxyJob(ctx, &wg); err != nil {
			log.ErrorMsg("Starting SOCKS proxy: %s", err)
		}
	}

	mst.startForegroundJob(ctx, &wg, cancel) // foreground job must cancel when it terminates

	go func() {
		for {
			m, err := mst.sess.Receive()
			if err != nil {
				if err == io.EOF {
					return
				}
				if ctx.Err() != nil {
					return // cancelled
				}

				log.ErrorMsg("Receiving next command: %s\n", err)
				continue
			}

			switch message := m.(type) {
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
