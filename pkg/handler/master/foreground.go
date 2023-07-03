package master

import (
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
	"sync"
)

func (mst *Master) startForegroundJob(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := mst.handleForeground(); err != nil {
			log.ErrorMsg("Running foreground job: %s", err)
		}
	}()
}

func (mst *Master) handleForeground() error {
	if err := mst.sess.Send(msg.Foreground{
		Exec: mst.mCfg.Exec,
		Pty:  mst.mCfg.Pty,
	}); err != nil {
		return fmt.Errorf("sess.Send(): %s", err)
	}

	conn, err := mst.sess.OpenNewChannel()
	if err != nil {
		return fmt.Errorf("OpenNewChannel() for conn: %s", err)
	}
	defer conn.Close()

	if mst.mCfg.LogFile != "" {
		var err error
		conn, err = log.NewLoggedConn(conn, mst.mCfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", mst.mCfg.LogFile, err)
		}
	}

	if mst.mCfg.Pty {
		connPtyCtl, err := mst.sess.OpenNewChannel()
		if err != nil {
			return fmt.Errorf("OpenNewChannel() for connPtyCtl: %s", err)
		}
		defer connPtyCtl.Close()

		if err := terminal.PipeWithPTY(connPtyCtl, conn, mst.cfg.Verbose); err != nil {
			return fmt.Errorf("terminal.PipeWithPTY(connCtl, connData): %s", err)
		}
	} else {
		terminal.Pipe(conn, mst.cfg.Verbose)
	}

	return nil
}
