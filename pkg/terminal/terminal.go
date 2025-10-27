// Package terminal provides utilities for terminal I/O piping with support
// for PTY (pseudo-terminal) mode, including raw mode and terminal size synchronization.
package terminal

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/pty"
	"dominicbreuker/goncat/pkg/semaphore"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/term"
)

// Pipe establishes bidirectional I/O between standard I/O and a network connection.
func Pipe(ctx context.Context, conn net.Conn, verbose bool, deps *config.Dependencies) {
	// Extract semaphore from deps if available
	var connSem *semaphore.ConnSemaphore
	if deps != nil && deps.ConnSem != nil {
		connSem = deps.ConnSem
	}

	stdio := pipeio.NewStdio(deps, connSem)

	// Acquire semaphore slot before starting I/O
	if err := stdio.AcquireSlot(ctx); err != nil {
		if verbose {
			log.ErrorMsg("Failed to acquire connection slot: %s\n", err)
		}
		return
	}

	pipeio.Pipe(ctx, stdio, conn, func(err error) {
		if verbose {
			log.ErrorMsg("Pipe(stdio, conn): %s\n", err)
		}
	})
}

// PipeWithPTY sets up a PTY-enabled connection between standard I/O and network connections.
// It puts the terminal in raw mode, pipes data, and synchronizes terminal size changes
// via connCtl. The terminal is restored to its original state when done.
func PipeWithPTY(ctx context.Context, connCtl, connData net.Conn, verbose bool, deps *config.Dependencies) error {
	log.InfoMsg("Enabling raw mode\n")
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("setting terminal to raw mode: %s", err)
	}

	defer func() {
		log.InfoMsg("Disabling raw mode\n")
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Printf("\033[2K\r") // clear line
	}()

	ctx, cancel := context.WithCancel(ctx)
	go syncTerminalSize(ctx, connCtl)

	Pipe(ctx, connData, verbose, deps)
	cancel()

	return nil
}

// syncTerminalSize continuously monitors the local terminal size and sends updates
// to the remote side via connCtl whenever the size changes.
func syncTerminalSize(ctx context.Context, connCtl net.Conn) {
	enc := gob.NewEncoder(connCtl)
	ticker := time.NewTicker(1 * time.Second)

	sizeRemote := pty.TerminalSize{}
	for {
		select {
		case <-ticker.C:
			size, err := pty.GetTerminalSize()
			if err != nil {
				log.ErrorMsg("can't identify terminal size: %s", err)
			}

			if size != sizeRemote {
				if err = enc.Encode(size); err != nil {
					log.ErrorMsg("can't send new Terminal size: %s", err)
					continue
				}
				sizeRemote = size
			}
		case <-ctx.Done():
			return
		}
	}
}
