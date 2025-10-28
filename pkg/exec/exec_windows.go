//go:build windows
// +build windows

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
)

// RunWithPTY executes the specified program in a ConPTY (Windows pseudo-console).
// It uses two connections: connCtl for terminal size synchronization and connData for I/O.
// The function blocks until both the program exits AND all I/O copying is complete.
func RunWithPTY(ctx context.Context, connCtl, connData net.Conn, program string, verbose bool, logger *log.Logger) error {
	cpty, err := pty.Create()
	if err != nil {
		return fmt.Errorf("failed to spawn a pty: %s", err)
	}
	defer cpty.Close()

	if err := cpty.Execute(program); err != nil {
		return fmt.Errorf("failed to run program: %s", err)
	}

	// Wait for both the command to exit and I/O copying to complete
	cmdDone := make(chan struct{})
	pipeDone := make(chan struct{})

	go func() {
		cpty.Wait()
		close(cmdDone)
	}()

	go syncTerminalSize(cpty, connCtl, verbose, logger)

	go func() {
		pipeio.Pipe(ctx, cpty, connData, func(err error) {
			if verbose {
				logger.ErrorMsg("Pipe(cpty, conn): %s\n", err)
			}
		})
		cpty.KillProcess()
		close(pipeDone)
	}()

	// Wait for both goroutines to complete
	<-cmdDone
	<-pipeDone

	return nil
}

// syncTerminalSize continuously reads terminal size updates from connCtl
// and applies them to the ConPTY.
func syncTerminalSize(cpty *pty.ConPTY, connCtl net.Conn, verbose bool, logger *log.Logger) {
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

		err = cpty.SetTerminalSize(*size)
		if err != nil {
			if verbose {
				logger.ErrorMsg("can't identify terminal size: %s", err)
			}
		}
	}
}

// TODO: recording is not good with pty enabled. maybe build replay feature like this? https://github.com/termbacktime/termbacktime/blob/master/cmd/live.go
