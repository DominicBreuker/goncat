package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/handler/portfwd"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
)

func (slv *Slave) handleConnectAsync(ctx context.Context, m msg.Connect) {
	go func() {
		h := portfwd.NewClient(ctx, m, slv.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("Running connect job: %s", err)
		}
	}()
}

func (slv *Slave) handlePortFwdAsync(ctx context.Context, m msg.PortFwd) {
	go func() {
		// Flip the settings, because remote port forwarding is like local port forwarding from the perspective of the slave
		cfg := portfwd.Config{
			LocalHost:  m.RemoteHost,
			LocalPort:  m.RemotePort,
			RemoteHost: m.LocalHost,
			RemotePort: m.LocalPort,
		}

		h := portfwd.NewServer(ctx, cfg, slv.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("Remote port forwarding: %s: %s\n", cfg, err)
		}
	}()
}
