package master

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
	"sync"
)

// startForegroundJob launches the foreground task in a goroutine and registers it with the wait group.
// The cancel function is called when the foreground job completes.
func (mst *master) startForegroundJob(ctx context.Context, wg *sync.WaitGroup, cancel func()) {
	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()

		if err := mst.handleForeground(ctx); err != nil {
			mst.cfg.Logger.ErrorMsg("Running foreground job: %s", err)
		}
	}()
}

// handleForeground manages the foreground connection, dispatching to the appropriate
// handler based on whether PTY is enabled.
func (mst *master) handleForeground(ctx context.Context) error {
	if mst.mCfg.Pty {
		return mst.handleForgroundPty(ctx)
	}

	return mst.handleForgroundPlain(ctx)
}

// handleForgroundPlain handles a foreground connection without PTY support,
// piping data between the session channel and the local terminal.
func (mst *master) handleForgroundPlain(ctx context.Context) error {
	m := msg.Foreground{
		Exec: mst.mCfg.Exec,
		Pty:  mst.mCfg.Pty,
	}

	conn, err := mst.sess.SendAndGetOneChannelContext(ctx, m)
	if err != nil {
		return fmt.Errorf("SendAndGetOneChannel(m): %s", err)
	}
	defer conn.Close()

	if mst.mCfg.LogFile != "" {
		var err error
		conn, err = log.NewLoggedConn(conn, mst.mCfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", mst.mCfg.LogFile, err)
		}
	}

	terminal.Pipe(ctx, conn, mst.cfg.Verbose, mst.cfg.Logger, mst.cfg.Deps)

	return nil
}

// handleForgroundPty handles a foreground connection with PTY support,
// managing both the data channel and PTY control channel.
func (mst *master) handleForgroundPty(ctx context.Context) error {
	m := msg.Foreground{
		Exec: mst.mCfg.Exec,
		Pty:  mst.mCfg.Pty,
	}

	connData, connPtyCtl, err := mst.sess.SendAndGetTwoChannelsContext(ctx, m)
	if err != nil {
		return fmt.Errorf("SendAndGetTwoChannels(m): %s", err)
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

	if err := terminal.PipeWithPTY(ctx, connPtyCtl, connData, mst.cfg.Verbose, mst.cfg.Logger, mst.cfg.Deps); err != nil {
		return fmt.Errorf("terminal.PipeWithPTY(connCtl, connData): %s", err)
	}

	return nil
}
