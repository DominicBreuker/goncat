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
func Run(ctx context.Context, conn net.Conn, program string, logger *log.Logger, deps *config.Dependencies) error {
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
	cmdDone := make(chan error, 1) // buffered, so goroutine won't block
	pipeDone := make(chan error, 1)

	go func() {
		cmdDone <- cmd.Wait()
	}()

	go func() {
		pipeDone <- pipeio.Pipe(ctx, cmdio, conn, func(err error) {
			logger.ErrorMsg("Run Pipe(pty, conn): %s\n", err)
		})
	}()

	// Wait for both goroutines to complete and collect errors.
	var cmdErr, pipeErr error
	for i := 0; i < 2; i++ {
		select {
		case err := <-cmdDone:
			cmdErr = err
		case err := <-pipeDone:
			pipeErr = err
		}
	}

	// Ensure the process is killed after I/O is done, not before.
	_ = cmd.Process().Kill() // Defensive: ensure cleanup

	if pipeErr != nil {
		return pipeErr
	}
	if cmdErr != nil {
		return cmdErr
	}
	return nil
}
