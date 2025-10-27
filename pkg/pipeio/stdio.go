package pipeio

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/semaphore"
	"io"
	"os"

	"github.com/muesli/cancelreader"
)

// Stdio provides a ReadWriteCloser interface for standard I/O streams.
// It uses cancelable reading from stdin when supported, allowing reads
// to be interrupted via Close.
type Stdio struct {
	stdin            io.Reader
	cancellableStdin cancelreader.CancelReader

	stdout io.Writer

	connSem *semaphore.ConnSemaphore // Connection semaphore for limiting concurrent connections
}

// NewStdio creates a new Stdio with cancelable stdin reading if supported by the platform.
// The deps parameter is optional and can be nil to use os.Stdin/os.Stdout.
// The connSem parameter is optional and can be nil if no connection limiting is needed.
// On platforms where cancelable reading is not supported, Read operations will use
// the provided stdin directly and cannot be interrupted via Close.
func NewStdio(deps *config.Dependencies, connSem *semaphore.ConnSemaphore) *Stdio {
	stdinFunc := config.GetStdinFunc(deps)
	stdoutFunc := config.GetStdoutFunc(deps)

	stdin := stdinFunc()
	stdout := stdoutFunc()

	out := Stdio{
		stdin:   stdin,
		stdout:  stdout,
		connSem: connSem,
	}

	// Try to create a cancelable reader if stdin is an os.File
	if stdinFile, ok := stdin.(*os.File); ok {
		cancellableStdin, err := cancelreader.NewReader(stdinFile)
		if err == nil {
			out.cancellableStdin = cancellableStdin
		}
	}

	return &out
}

// AcquireSlot attempts to acquire the connection semaphore.
// Must be called before starting I/O operations.
// Returns error if timeout or context cancellation occurs.
func (s *Stdio) AcquireSlot(ctx context.Context) error {
	if s.connSem == nil {
		return nil
	}
	return s.connSem.Acquire(ctx)
}

// Read reads from stdin, using the cancelable reader if available.
func (s *Stdio) Read(p []byte) (n int, err error) {
	if s.cancellableStdin != nil {
		return s.cancellableStdin.Read(p)
	}

	return s.stdin.Read(p)
}

// Write writes to stdout.
func (s *Stdio) Write(p []byte) (n int, err error) {
	return s.stdout.Write(p)
}

// Close cancels any pending reads from stdin if using a cancelable reader
// and releases the connection semaphore.
func (s *Stdio) Close() error {
	if s.connSem != nil {
		s.connSem.Release()
	}
	if s.cancellableStdin != nil {
		s.cancellableStdin.Cancel()
	}
	return nil
}
