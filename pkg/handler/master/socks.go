package master

import (
	"context"
	socksmaster "dominicbreuker/goncat/pkg/handler/socks/master"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"sync"
)

// startSocksProxyJob starts a SOCKS5 proxy server in a goroutine.
// The proxy listens for SOCKS5 client connections and forwards requests
// to the slave through the multiplexed session.
func (mst *Master) startSocksProxyJob(ctx context.Context, wg *sync.WaitGroup) error {
	cfg := socksmaster.Config{
		LocalHost: mst.mCfg.Socks.Host,
		LocalPort: mst.mCfg.Socks.Port,
	}
	srv, err := socksmaster.NewServer(ctx, cfg, mst.sess)
	if err != nil {
		return fmt.Errorf("creating server: %s", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := srv.Serve(); err != nil {
			log.ErrorMsg("SOCKS: %s: %s\n", cfg, err)
		}
	}()

	return nil
}
