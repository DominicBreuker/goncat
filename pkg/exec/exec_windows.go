//go:build windows
// +build windows

package exec

import (
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/pty"
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

func RunWithPTY(connCtl, connData net.Conn, program string, verbose bool) error {
	cpty, err := pty.Create()
	if err != nil {
		return fmt.Errorf("failed to spawn a pty: %s", err)
	}
	defer cpty.Close()

	if err := cpty.Execute(program); err != nil {
		return fmt.Errorf("failed to run program: %s", err)
	}

	done := make(chan struct{})

	go func() {
		cpty.Wait()
		done <- struct{}{}
	}()

	go syncTerminalSize(cpty, connCtl, verbose)

	go func() {
		pipeio.Pipe(cpty, connData, func(err error) {
			if verbose {
				log.ErrorMsg("Pipe(cpty, conn): %s\n", err)
			}
		})
		cpty.KillProcess()
		done <- struct{}{}
	}()
	<-done

	return nil
}

func syncTerminalSize(cpty *pty.ConPTY, connCtl net.Conn, verbose bool) {
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

		err = cpty.SetTerminalSize(*size)
		if err != nil {
			if verbose {
				log.ErrorMsg("can't identify terminal size: %s", err)
			}
		}
	}
}

// TODO: recording is not good with pty enabled. maybe build replay feature like this? https://github.com/termbacktime/termbacktime/blob/master/cmd/live.go
