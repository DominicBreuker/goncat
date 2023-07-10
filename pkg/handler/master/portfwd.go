package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/portfwd"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"sync"
)

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

func (mst *Master) handleConnectAsync(ctx context.Context, m msg.Connect) {
	go func() {
		h := portfwd.NewClient(ctx, m, mst.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("Running connect job: %s", err)
		}
	}()
}
