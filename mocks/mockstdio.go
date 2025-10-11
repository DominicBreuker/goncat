// Package mocks provides mock implementations for testing.
package mocks

import (
	"bytes"
	"io"
	"sync"
)

// MockStdio provides mock implementations of stdin and stdout for testing.
// It uses pipes internally to allow proper bidirectional communication.
type MockStdio struct {
	stdinReader  *io.PipeReader
	stdinWriter  *io.PipeWriter
	stdoutReader *io.PipeReader
	stdoutWriter *io.PipeWriter
	outputBuf    *bytes.Buffer
	mu           sync.Mutex
}

// NewMockStdio creates a new mock stdio with pipe-based streams.
func NewMockStdio() *MockStdio {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	m := &MockStdio{
		stdinReader:  stdinR,
		stdinWriter:  stdinW,
		stdoutReader: stdoutR,
		stdoutWriter: stdoutW,
		outputBuf:    &bytes.Buffer{},
	}

	// Start goroutine to collect stdout data
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutR.Read(buf)
			if n > 0 {
				m.mu.Lock()
				m.outputBuf.Write(buf[:n])
				m.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	return m
}

// WriteToStdin writes data to the mock stdin pipe.
// This simulates user input that will be read by the application.
func (m *MockStdio) WriteToStdin(data []byte) (int, error) {
	return m.stdinWriter.Write(data)
}

// ReadFromStdout reads data from the mock stdout buffer.
// This retrieves what the application has written to stdout.
func (m *MockStdio) ReadFromStdout() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.outputBuf.String()
}

// GetStdin returns a reader for stdin (used by the dependency injection).
func (m *MockStdio) GetStdin() io.Reader {
	return m.stdinReader
}

// GetStdout returns a writer for stdout (used by the dependency injection).
func (m *MockStdio) GetStdout() io.Writer {
	return m.stdoutWriter
}

// Close closes the mock stdio pipes.
func (m *MockStdio) Close() error {
	m.stdinWriter.Close()
	m.stdoutWriter.Close()
	return nil
}
