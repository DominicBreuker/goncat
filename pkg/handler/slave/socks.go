package slave

import (
	"context"
	socksslave "dominicbreuker/goncat/pkg/handler/socks/slave"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
)

func (slv *Slave) handleSocksConnectAsync(ctx context.Context, m msg.SocksConnect) {
	go func() {
		tr := socksslave.NewTCPRelay(ctx, m, slv.sess)
		if err := tr.Handle(); err != nil {
			log.ErrorMsg("Running SocksConnect job: %s\n", err)
		}
	}()
}

func (slv *Slave) handleSocksAsociateAsync(ctx context.Context, _ msg.SocksAssociate) {
	go func() {
		relay, err := socksslave.NewUDPRelay(ctx, slv.sess)
		if err != nil {
			log.ErrorMsg("Running SocksAssociate job: %s\n", err)
			return
		}
		defer relay.Close()

		if err := relay.Serve(); err != nil {
			log.ErrorMsg("UDP Relay: %s\n", err)
		}
	}()
}
