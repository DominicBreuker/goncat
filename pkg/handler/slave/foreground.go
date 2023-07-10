package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/exec"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
)

func (slv *Slave) handleForegroundAsync(ctx context.Context, m msg.Foreground) {
	go func() {
		if err := slv.handleForeground(ctx, m); err != nil {
			log.ErrorMsg("Running foreground job: %s", err)
		}
	}()
}

func (slv *Slave) handleForeground(ctx context.Context, m msg.Foreground) error {
	conn, err := slv.sess.AcceptNewChannel()
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %s", err)
	}
	defer conn.Close()

	if m.Exec == "" {
		terminal.Pipe(ctx, conn, slv.cfg.Verbose)
	} else {
		if m.Pty {
			connPtyCtl, err := slv.sess.AcceptNewChannel()
			if err != nil {
				return fmt.Errorf("AcceptNewChannel() for connPtyCtl: %s", err)
			}
			defer connPtyCtl.Close()

			if err := exec.RunWithPTY(ctx, connPtyCtl, conn, m.Exec, slv.cfg.Verbose); err != nil {
				return fmt.Errorf("exec.RunWithPTY(...): %s", err)
			}
		} else {
			if err := exec.Run(ctx, conn, m.Exec); err != nil {
				return fmt.Errorf("exec.Run(conn, %s): %s", m.Exec, err)
			}
		}
	}

	return nil
}
