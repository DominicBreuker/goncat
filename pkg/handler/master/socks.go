package master

import (
	"context"
	"dominicbreuker/goncat/pkg/handler/socks"
	"dominicbreuker/goncat/pkg/log"
	"sync"
)

func (mst *Master) startSocksProxyJob(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		cfg := socks.Config{
			LocalHost: mst.mCfg.Socks.Host,
			LocalPort: mst.mCfg.Socks.Port,
		}
		h := socks.NewServer(ctx, cfg, mst.sess)
		if err := h.Handle(); err != nil {
			log.ErrorMsg("SOCKS: %s: %s\n", cfg, err)
		}
	}()
}
