package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/exec"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
)

// handleForegroundAsync handles a foreground task request from the master asynchronously.
// When the foreground handler completes, it closes the slave connection to terminate the session.
func (slv *slave) handleForegroundAsync(ctx context.Context, m msg.Foreground) {
	go func() {
		if err := slv.handleForeground(ctx, m); err != nil {
			slv.cfg.Logger.ErrorMsg("Running foreground job: %s", err)
		}
		slv.Close()
	}()
}

// handleForeground processes a foreground task request, either starting an interactive
// shell or executing a command with or without PTY support.
func (slv *slave) handleForeground(ctx context.Context, m msg.Foreground) error {
	conn, err := slv.sess.AcceptNewChannelContext(ctx)
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %s", err)
	}
	defer conn.Close()

	if m.Exec == "" {
		terminal.Pipe(ctx, conn, slv.cfg.Verbose, slv.cfg.Logger, slv.cfg.Deps)
	} else {
		if m.Pty {
			connPtyCtl, err := slv.sess.AcceptNewChannelContext(ctx)
			if err != nil {
				return fmt.Errorf("AcceptNewChannel() for connPtyCtl: %s", err)
			}
			defer connPtyCtl.Close()

			if err := exec.RunWithPTY(ctx, connPtyCtl, conn, m.Exec, slv.cfg.Verbose, slv.cfg.Logger); err != nil {
				return fmt.Errorf("exec.RunWithPTY(...): %s", err)
			}
		} else {
			if err := exec.Run(ctx, conn, m.Exec, slv.cfg.Logger, slv.cfg.Deps); err != nil {
				return fmt.Errorf("exec.Run(conn, %s): %s", m.Exec, err)
			}
		}
	}

	return nil
}
