// Package exec provides functionality for executing programs
// and connecting their I/O to network connections, with support
// for both plain execution and PTY (pseudo-terminal) mode.
package exec

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/pipeio"
	"fmt"
	"net"
)

// Run executes the specified program and pipes its stdin/stdout/stderr
// to and from the provided network connection. The function blocks until
// both the program exits AND all I/O copying is complete.
func Run(ctx context.Context, conn net.Conn, program string, deps *config.Dependencies) error {
	execCmd := config.GetExecCommandFunc(deps)
	cmd := execCmd(program)

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

	// Wait for both the command to exit and I/O copying to complete
	cmdDone := make(chan struct{})
	pipeDone := make(chan struct{})

	go func() {
		cmd.Wait()
		close(cmdDone)
	}()

	go func() {
		pipeio.Pipe(ctx, cmdio, conn, func(err error) {
			log.ErrorMsg("Run Pipe(pty, conn): %s\n", err)
		})
		cmd.Process().Kill()
		close(pipeDone)
	}()

	// Wait for both goroutines to complete
	<-cmdDone
	<-pipeDone

	return nil
}
