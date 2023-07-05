package master

import (
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
	"sync"
)

func (mst *Master) startForegroundJob(wg *sync.WaitGroup, cancel func()) {
	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()

		if err := mst.handleForeground(); err != nil {
			log.ErrorMsg("Running foreground job: %s", err)
		}
	}()
}

func (mst *Master) handleForeground() error {
	if mst.mCfg.Pty {
		return mst.handleForgroundPty()
	}

	return mst.handleForgroundPlain()
}

func (mst *Master) handleForgroundPlain() error {
	m := msg.Foreground{
		Exec: mst.mCfg.Exec,
		Pty:  mst.mCfg.Pty,
	}

	conn, err := mst.sess.SendAndOpenOneChannel(m)
	if err != nil {
		return fmt.Errorf("SendAndOpenOneChannel(m): %s", err)
	}
	defer conn.Close()

	if mst.mCfg.LogFile != "" {
		var err error
		conn, err = log.NewLoggedConn(conn, mst.mCfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", mst.mCfg.LogFile, err)
		}
	}

	terminal.Pipe(conn, mst.cfg.Verbose)

	return nil
}

func (mst *Master) handleForgroundPty() error {
	m := msg.Foreground{
		Exec: mst.mCfg.Exec,
		Pty:  mst.mCfg.Pty,
	}

	connData, connPtyCtl, err := mst.sess.SendAndOpenTwoChannels(m)
	if err != nil {
		return fmt.Errorf("SendAndOpenTwoChannels(m): %s", err)
	}
	defer connData.Close()
	defer connPtyCtl.Close()

	if mst.mCfg.LogFile != "" {
		var err error
		connData, err = log.NewLoggedConn(connData, mst.mCfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", mst.mCfg.LogFile, err)
		}
	}

	if err := terminal.PipeWithPTY(connPtyCtl, connData, mst.cfg.Verbose); err != nil {
		return fmt.Errorf("terminal.PipeWithPTY(connCtl, connData): %s", err)
	}

	return nil
}
