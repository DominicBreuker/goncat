package pipeio

import (
	"os"

	"github.com/muesli/cancelreader"
)

// Stdio is a ReadWriteCloser on standard in and out
// if possible, it ensures that Reads to stdin are cancelled when it gets closed
type Stdio struct {
	stdin            *os.File
	cancellableStdin cancelreader.CancelReader

	stdout *os.File
}

// NewStdio sets up a new Stdio with cancellable reader in stdin if supported
// func NewStdio() *Stdio {
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

// Read reads from stdin
func (s *Stdio) Read(p []byte) (n int, err error) {
	if s.cancellableStdin != nil {
		return s.cancellableStdin.Read(p)
	}

	return s.stdin.Read(p)
}

// Write writes to stdout
func (s *Stdio) Write(p []byte) (n int, err error) {
	return s.stdout.Write(p)
}

// Close cancels reads from stdin if possible
func (s *Stdio) Close() error {
	if s.cancellableStdin != nil {
		s.cancellableStdin.Cancel()
	}
	return nil
}
