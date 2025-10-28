package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/portfwd"
	"dominicbreuker/goncat/pkg/mux/msg"
	"sync"
)

// startLocalPortFwdJob starts a local port forwarding job in a goroutine.
// It sets up a server to accept connections on the local port and forwards them
// to the remote destination through the multiplexed session.
func (mst *master) startLocalPortFwdJob(ctx context.Context, wg *sync.WaitGroup, lpf *config.LocalPortForwardingCfg) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		cfg := portfwd.Config{
			Protocol:   lpf.Protocol,
			LocalHost:  lpf.LocalHost,
			LocalPort:  lpf.LocalPort,
			RemoteHost: lpf.RemoteHost,
			RemotePort: lpf.RemotePort,
			Timeout:    mst.cfg.Timeout,
			Logger:     mst.cfg.Logger,
		}
		h := portfwd.NewServer(ctx, cfg, mst.sess, mst.cfg.Deps)
		if err := h.Handle(); err != nil {
			mst.cfg.Logger.ErrorMsg("Local port forwarding: %s: %s\n", lpf, err)
		}
	}()
}

// startRemotePortFwdJob starts a remote port forwarding job in a goroutine.
// It sends a port forwarding message to the slave, instructing it to listen
// on the remote port and forward connections back to the master.
func (mst *master) startRemotePortFwdJob(ctx context.Context, wg *sync.WaitGroup, rpf *config.RemotePortForwardingCfg) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		m := msg.PortFwd{
			Protocol:   rpf.Protocol,
			LocalHost:  rpf.LocalHost,
			LocalPort:  rpf.LocalPort,
			RemoteHost: rpf.RemoteHost,
			RemotePort: rpf.RemotePort,
		}

		if err := mst.sess.SendContext(ctx, m); err != nil {
			mst.cfg.Logger.ErrorMsg("Setting up remote port forwarding: %s: %s\n", rpf, err)
		}
	}()
}

// handleConnectAsync handles an incoming Connect message from the slave asynchronously.
// This is used for remote port forwarding when the slave forwards a connection back to the master.
func (mst *master) handleConnectAsync(ctx context.Context, m msg.Connect) {
	go func() {
		h := portfwd.NewClient(ctx, m, mst.sess, mst.cfg.Logger, mst.cfg.Deps)
		if err := h.Handle(); err != nil {
			mst.cfg.Logger.ErrorMsg("Running connect job: %s", err)
		}
	}()
}
