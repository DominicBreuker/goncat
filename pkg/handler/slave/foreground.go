package slave

import (
	"dominicbreuker/goncat/pkg/exec"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
)

func (slv *Slave) handleForegroundAsync(m msg.Foreground) {
	go func() {
		if err := slv.handleForeground(m); err != nil {
			log.ErrorMsg("Running foreground job: %s", err)
		}
	}()
}

func (slv *Slave) handleForeground(m msg.Foreground) error {
	conn, err := slv.sess.AcceptNewChannel()
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %s", err)
	}
	defer conn.Close()

	if m.Exec == "" {
		terminal.Pipe(conn, slv.cfg.Verbose)
	} else {
		if m.Pty {
			connPtyCtl, err := slv.sess.AcceptNewChannel()
			if err != nil {
				return fmt.Errorf("AcceptNewChannel() for connPtyCtl: %s", err)
			}
			defer connPtyCtl.Close()

			if err := exec.RunWithPTY(connPtyCtl, conn, m.Exec, slv.cfg.Verbose); err != nil {
				return fmt.Errorf("exec.RunWithPTY(...): %s", err)
			}
		} else {
			if err := exec.Run(conn, m.Exec); err != nil {
				return fmt.Errorf("exec.Run(conn, %s): %s", m.Exec, err)
			}
		}
	}

	return nil
}
