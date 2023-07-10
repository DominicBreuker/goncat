//go:build !windows
// +build !windows

package exec

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/pty"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"syscall"
)

// RunWithPTY ...
func RunWithPTY(ctx context.Context, connCtl, connData net.Conn, program string, verbose bool) error {
	cmd := exec.Command(program)

	pty, tty, err := pty.NewPty()
	if err != nil {
		return fmt.Errorf("starting pty: %s", err)
	}
	defer tty.Close()
	defer pty.Close()

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setctty: true,
		Setsid:  true,
	}

	cmd.Stdout = tty
	cmd.Stdin = tty
	cmd.Stderr = tty

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Run(): %s", err)
	}

	done := make(chan struct{})

	go func() {
		cmd.Wait()
		done <- struct{}{}
	}()

	go syncTerminalSize(pty, connCtl, verbose)

	go func() {
		pipeio.Pipe(ctx, pty, connData, func(err error) {
			if verbose {
				log.ErrorMsg("Pipe(pty, conn): %s\n", err)
			}
		})
		cmd.Process.Kill()
		done <- struct{}{}
	}()
	<-done

	return nil
}

func syncTerminalSize(ptyFd *os.File, connCtl net.Conn, verbose bool) {
	dec := gob.NewDecoder(connCtl)

	for {
		size := &pty.TerminalSize{}
		err := dec.Decode(size)
		if size == nil || err != nil {
			if err == io.EOF {
				return
			}

			if verbose {
				log.ErrorMsg("can't decode new Terminal size: %s", err)
			}
		}

		err = pty.SetTerminalSize(ptyFd, *size)
		if err != nil {
			if verbose {
				log.ErrorMsg("can't identify terminal size: %s", err)
			}
		}
	}
}
