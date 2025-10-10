package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/portfwd"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"sync"
)

// startLocalPortFwdJobJob starts a local port forwarding job in a goroutine.
// It sets up a server to accept connections on the local port and forwards them
// to the remote destination through the multiplexed session.
func (mst *Master) startLocalPortFwdJobJob(ctx context.Context, wg *sync.WaitGroup, lpf *config.LocalPortForwardingCfg) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		cfg := portfwd.Config{
			LocalHost:  lpf.LocalHost,
			LocalPort:  lpf.LocalPort,
			RemoteHost: lpf.RemoteHost,
			RemotePort: lpf.RemotePort,
		}
		h := portfwd.NewServer(ctx, cfg, mst.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("Local port forwarding: %s: %s\n", lpf, err)
		}
	}()
}

// startRemotePortFwdJobJob starts a remote port forwarding job in a goroutine.
// It sends a port forwarding message to the slave, instructing it to listen
// on the remote port and forward connections back to the master.
func (mst *Master) startRemotePortFwdJobJob(ctx context.Context, wg *sync.WaitGroup, rpf *config.RemotePortForwardingCfg) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		m := msg.PortFwd{
			LocalHost:  rpf.LocalHost,
			LocalPort:  rpf.LocalPort,
			RemoteHost: rpf.RemoteHost,
			RemotePort: rpf.RemotePort,
		}

		if err := mst.sess.Send(m); err != nil {
			log.ErrorMsg("Setting up remote port forwarding: %s: %s\n", rpf, err)
		}
	}()
}

// handleConnectAsync handles an incoming Connect message from the slave asynchronously.
// This is used for remote port forwarding when the slave forwards a connection back to the master.
func (mst *Master) handleConnectAsync(ctx context.Context, m msg.Connect) {
	go func() {
		h := portfwd.NewClient(ctx, m, mst.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("Running connect job: %s", err)
		}
	}()
}
