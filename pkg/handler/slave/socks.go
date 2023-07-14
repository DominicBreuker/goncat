package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/handler/socks"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
)

func (slv *Slave) handleSocksConnectAsync(ctx context.Context, m msg.SocksConnect) {
	go func() {
		h := socks.NewClient(ctx, m, slv.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("Running SocksConnect job: %s\n", err)
		}
	}()
}
