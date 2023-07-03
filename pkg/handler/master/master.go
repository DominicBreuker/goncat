package master

import (
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux"
	"fmt"
	"net"
	"sync"
)

// Master ...
type Master struct {
	cfg  *config.Shared
	mCfg *config.Master

	sess *mux.MasterSession
}

// New ...
func New(cfg *config.Shared, mCfg *config.Master, conn net.Conn) (*Master, error) {
	sess, err := mux.OpenSession(conn)
	if err != nil {
		return nil, fmt.Errorf("mux.OpenSession(conn): %s", err)
	}

	return &Master{
		cfg:  cfg,
		mCfg: mCfg,
		sess: sess,
	}, nil
}

// Close ...
func (mst *Master) Close() error {
	return mst.sess.Close()
}

// Handle ...
func (mst *Master) Handle() error {
	var wg sync.WaitGroup

	mst.startForegroundJob(&wg)

	wg.Wait()

	return nil
}
