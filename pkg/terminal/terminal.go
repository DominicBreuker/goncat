package terminal

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/pty"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/term"
)

// Pipe ...
func Pipe(ctx context.Context, conn net.Conn, verbose bool) {
	pipeio.Pipe(ctx, pipeio.NewStdio(), conn, func(err error) {
		if verbose {
			log.ErrorMsg("Pipe(stdio, conn): %s\n", err)
		}
	})
}

// PipeWithPTY ...
func PipeWithPTY(ctx context.Context, connCtl, connData net.Conn, verbose bool) error {
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

	Pipe(ctx, connData, verbose)
	cancel()

	return nil
}

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
