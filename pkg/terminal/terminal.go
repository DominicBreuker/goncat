package terminal

import (
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

func Pipe(conn net.Conn, verbose bool) {
	pipeio.Pipe(pipeio.Stdio, conn, func(err error) {
		if verbose {
			log.ErrorMsg("Pipe(stdio, conn): %s\n", err)
		}
	})
}

func PipeWithPTY(connCtl, connData net.Conn, verbose bool) error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("setting terminal to raw mode: %s", err)
	}
	defer func() {
		log.InfoMsg("Disabling raw mode\n")
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Printf("\033[2K\r") // clear line
	}()

	go syncTerminalSize(connCtl)
	Pipe(connData, verbose)

	return nil
}

func syncTerminalSize(connCtl net.Conn) {
	enc := gob.NewEncoder(connCtl)

	sizeRemote := pty.TerminalSize{}
	for {
		time.Sleep(1 * time.Second)
		size, err := pty.GetTerminalSize()
		if err != nil {
			log.ErrorMsg("can't identify terminal size: %s", err)
		}

		if size != sizeRemote {
			if err = enc.Encode(size); err != nil {
				log.ErrorMsg("can't send new Terminal size: %s", err)
			}
			sizeRemote = size
		}
	}
}
