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

// RunWithPTY executes the specified program in a PTY (pseudo-terminal) on Unix systems.
// It uses two connections: connCtl for terminal size synchronization and connData for I/O.
// The function blocks until both the program exits AND all I/O copying is complete.
func RunWithPTY(ctx context.Context, connCtl, connData net.Conn, program string, verbose bool, logger *log.Logger) error {
	cmd := exec.Command(program)

	pty, tty, err := pty.NewPty()
	if err != nil {
		return fmt.Errorf("starting pty: %s", err)
	}

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

	// Wait for both the command to exit and I/O copying to complete
	cmdDone := make(chan struct{})
	pipeDone := make(chan struct{})

	go func() {
		cmd.Wait()
		tty.Close()
		close(cmdDone)
	}()

	go syncTerminalSize(pty, connCtl, verbose, logger)

	go func() {
		pipeio.Pipe(ctx, pty, connData, func(err error) {
			if verbose {
				logger.ErrorMsg("Pipe(pty, conn): %s\n", err)
			}
		})
		cmd.Process.Kill()
		pty.Close()
		close(pipeDone)
	}()

	// Wait for both goroutines to complete
	<-cmdDone
	<-pipeDone

	return nil
}

// syncTerminalSize continuously reads terminal size updates from connCtl
// and applies them to the PTY file descriptor.
func syncTerminalSize(ptyFd *os.File, connCtl net.Conn, verbose bool, logger *log.Logger) {
	dec := gob.NewDecoder(connCtl)

	for {
		size := &pty.TerminalSize{}
		err := dec.Decode(size)
		if size == nil || err != nil {
			if err == io.EOF {
				return
			}

			if verbose {
				logger.ErrorMsg("can't decode new Terminal size: %s", err)
			}
		}

		err = pty.SetTerminalSize(ptyFd, *size)
		if err != nil {
			if verbose {
				logger.ErrorMsg("can't identify terminal size: %s", err)
			}
		}
	}
}
