package pipeio

import (
	"os"

	"github.com/muesli/cancelreader"
)

// Stdio provides a ReadWriteCloser interface for standard I/O streams.
// It uses cancelable reading from stdin when supported, allowing reads
// to be interrupted via Close.
type Stdio struct {
	stdin            *os.File
	cancellableStdin cancelreader.CancelReader

	stdout *os.File
}

// NewStdio creates a new Stdio with cancelable stdin reading if supported by the platform.
func NewStdio() *Stdio {
	out := Stdio{
		stdin:  os.Stdin,
		stdout: os.Stdout,
	}

	cancellableStdin, err := (cancelreader.NewReader(os.Stdin))
	if err != nil {
		return &out
	}

	out.cancellableStdin = cancellableStdin
	return &out
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

// Close cancels any pending reads from stdin if using a cancelable reader.
func (s *Stdio) Close() error {
	if s.cancellableStdin != nil {
		s.cancellableStdin.Cancel()
	}
	return nil
}
