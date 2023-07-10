package exec

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/pipeio"
	"fmt"
	"net"
	"os/exec"
)

// Run ...
func Run(ctx context.Context, conn net.Conn, program string) error {
	cmd := exec.Command(program)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdoutPipe(): %s", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdinPipe(): %s", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cmd.StderrPipe(): %s", err)
	}

	cmdio := pipeio.NewCmdio(stdout, stderr, stdin)

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Run(): %s", err)
	}

	done := make(chan struct{})

	go func() {
		cmd.Wait()
		done <- struct{}{}
	}()

	go func() {
		pipeio.Pipe(ctx, cmdio, conn, func(err error) {
			log.ErrorMsg("Run Pipe(pty, conn): %s\n", err)
		})
		cmd.Process.Kill()
		done <- struct{}{}
	}()
	<-done

	return nil
}
