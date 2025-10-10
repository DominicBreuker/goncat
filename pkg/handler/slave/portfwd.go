package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/handler/portfwd"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
)

// handleConnectAsync handles a Connect message from the master asynchronously.
// It establishes a connection to the requested destination and pipes data.
func (slv *Slave) handleConnectAsync(ctx context.Context, m msg.Connect) {
	go func() {
		h := portfwd.NewClient(ctx, m, slv.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("Running connect job: %s", err)
		}
	}()
}

// handlePortFwdAsync handles a remote port forwarding request from the master asynchronously.
// From the slave's perspective, remote port forwarding is like local port forwarding,
// so it listens on the remote port and forwards connections to the local destination.
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
