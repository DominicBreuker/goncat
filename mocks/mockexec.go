// Package mocks provides mock implementations for testing.
package mocks

import (
	"dominicbreuker/goncat/pkg/config"
	"fmt"
	"io"
	"strings"
	"sync"
)

// MockExec provides a mock implementation of command execution for testing.
// It simulates running a command like /bin/sh, responding to specific commands.
type MockExec struct{}

// NewMockExec creates a new mock exec that simulates shell behavior.
func NewMockExec() *MockExec {
	return &MockExec{}
}

// Command returns a mock command that simulates /bin/sh behavior.
// It responds to specific commands like "echo" and "whoami", and rejects others.
func (m *MockExec) Command(program string) config.Cmd {
	return &mockCmd{
		program: program,
		exec:    m,
	}
}

// mockCmd implements config.Cmd interface.
type mockCmd struct {
	program     string
	exec        *MockExec
	stdinPipe   *io.PipeWriter
	stdinRead   *io.PipeReader
	stdoutPipe  *io.PipeReader
	stdoutWrite *io.PipeWriter
	stderrPipe  *io.PipeReader
	stderrWrite *io.PipeWriter
	started     bool
	finished    bool
	mu          sync.Mutex
}

func (m *mockCmd) StdoutPipe() (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil, fmt.Errorf("StdoutPipe called after Start")
	}

	r, w := io.Pipe()
	m.stdoutPipe = r
	m.stdoutWrite = w
	return r, nil
}

func (m *mockCmd) StdinPipe() (io.WriteCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil, fmt.Errorf("StdinPipe called after Start")
	}

	r, w := io.Pipe()
	m.stdinPipe = w
	m.stdinRead = r
	return w, nil
}

func (m *mockCmd) StderrPipe() (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil, fmt.Errorf("StderrPipe called after Start")
	}

	r, w := io.Pipe()
	m.stderrPipe = r
	m.stderrWrite = w
	return r, nil
}

func (m *mockCmd) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("command already started")
	}

	m.started = true

	// Start a goroutine to simulate shell behavior
	go func() {
		defer m.stdoutWrite.Close()
		defer m.stderrWrite.Close()

		// Process commands line by line
		buf := make([]byte, 4096)
		var line []byte
		for {
			n, err := m.stdinRead.Read(buf)
			if n > 0 {
				line = append(line, buf[:n]...)
				// Process complete lines
				for len(line) > 0 {
					idx := -1
					for i := 0; i < len(line); i++ {
						if line[i] == '\n' {
							idx = i
							break
						}
					}
					if idx == -1 {
						break // No complete line yet
					}

					// Process the line
					cmd := string(line[:idx])
					line = line[idx+1:]
					m.processCommand(cmd)
				}
			}
			if err != nil {
				break
			}
		}

		m.mu.Lock()
		m.finished = true
		m.mu.Unlock()
	}()

	return nil
}

// processCommand simulates shell command execution for /bin/sh.
func (m *mockCmd) processCommand(cmd string) {
	// Trim whitespace
	cmd = strings.TrimSpace(cmd)

	if cmd == "" {
		return
	}

	// Check if command starts with "echo "
	if strings.HasPrefix(cmd, "echo ") {
		// Extract the echo argument and write it to stdout
		arg := cmd[5:] // Remove "echo "
		m.stdoutWrite.Write([]byte(arg + "\n"))
	} else if cmd == "whoami" {
		// Respond with mock shell identifier
		m.stdoutWrite.Write([]byte("mockcmd[" + m.program + "]\n"))
	} else if cmd == "exit" {
		// Close stdin to signal exit
		m.stdinRead.Close()
	} else {
		// Unsupported command
		m.stderrWrite.Write([]byte("command not supported by mock: " + cmd + "\n"))
	}
}

func (m *mockCmd) Wait() error {
	// Wait for the command to finish
	for {
		m.mu.Lock()
		finished := m.finished
		m.mu.Unlock()

		if finished {
			break
		}
		// Small sleep to avoid busy waiting
		// In a real implementation, we would use proper synchronization
	}
	return nil
}

func (m *mockCmd) Process() config.Process {
	return &mockProcess{cmd: m}
}

// mockProcess implements config.Process interface.
type mockProcess struct {
	cmd *mockCmd
}

func (m *mockProcess) Kill() error {
	m.cmd.mu.Lock()
	defer m.cmd.mu.Unlock()

	// Close pipes to signal termination
	if m.cmd.stdinRead != nil {
		m.cmd.stdinRead.Close()
	}
	if m.cmd.stdoutWrite != nil {
		m.cmd.stdoutWrite.Close()
	}
	if m.cmd.stderrWrite != nil {
		m.cmd.stderrWrite.Close()
	}

	m.cmd.finished = true
	return nil
}
