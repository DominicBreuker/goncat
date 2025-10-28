package slave

import (
	"context"
	socksslave "dominicbreuker/goncat/pkg/handler/socks/slave"
	"dominicbreuker/goncat/pkg/mux/msg"
)

// handleSocksConnectAsync handles a SOCKS5 CONNECT request from the master asynchronously.
// It establishes a TCP connection to the requested destination.
func (slv *slave) handleSocksConnectAsync(ctx context.Context, m msg.SocksConnect) {
	go func() {
		tr := socksslave.NewTCPRelay(ctx, m, slv.sess, slv.cfg.Logger, slv.cfg.Deps)
		if err := tr.Handle(); err != nil {
			slv.cfg.Logger.ErrorMsg("Running SocksConnect job: %s\n", err)
		}
	}()
}

// handleSocksAsociateAsync handles a SOCKS5 ASSOCIATE request from the master asynchronously.
// It creates a UDP relay to handle UDP datagrams for the SOCKS5 client.
func (slv *slave) handleSocksAsociateAsync(ctx context.Context, _ msg.SocksAssociate) {
	go func() {
		relay, err := socksslave.NewUDPRelay(ctx, slv.sess, slv.cfg.Logger, slv.cfg.Deps)
		if err != nil {
			slv.cfg.Logger.ErrorMsg("Running SocksAssociate job: %s\n", err)
			return
		}
		defer relay.Close()

		if err := relay.Serve(); err != nil {
			slv.cfg.Logger.ErrorMsg("UDP Relay: %s\n", err)
		}
	}()
}
