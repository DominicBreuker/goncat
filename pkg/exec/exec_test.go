package exec

import (
	"bytes"
	"context"
	"dominicbreuker/goncat/mocks"
	"dominicbreuker/goncat/pkg/config"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeConn is a minimal fake implementation of net.Conn for testing.
type fakeConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

// waitForOutput waits for the expected string to appear in the write buffer by polling
func (f *fakeConn) waitForOutput(expected string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for {
		// Check if the expected string is in the buffer
		if strings.Contains(f.writeBuf.String(), expected) {
			return nil
		}
		
		// Check if we've exceeded the timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for output %q, got: %q", expected, f.writeBuf.String())
		}
		
		// Wait a bit before checking again
		time.Sleep(10 * time.Millisecond)
	}
}

func (f *fakeConn) Read(p []byte) (n int, err error) {
	if f.closed {
		return 0, io.EOF
	}
	return f.readBuf.Read(p)
}

func (f *fakeConn) Write(p []byte) (n int, err error) {
	if f.closed {
		return 0, io.ErrClosedPipe
	}
	return f.writeBuf.Write(p)
}

func (f *fakeConn) Close() error {
	f.closed = true
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr                { return nil }
func (f *fakeConn) RemoteAddr() net.Addr               { return nil }
func (f *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (f *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (f *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

func TestRun_Echo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mockExec := mocks.NewMockExec()
	deps := &config.Dependencies{
		ExecCommand: mockExec.Command,
	}

	conn := newFakeConn()

	// Prepare input for the mock shell
	conn.readBuf.WriteString("echo hello world\n")
	conn.readBuf.WriteString("exit\n")

	// Run the command in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, conn, "/bin/sh", deps)
	}()

	// Wait for expected output to appear (Run() may return before pipeio.Pipe finishes copying)
	if err := conn.waitForOutput("hello world", 2*time.Second); err != nil {
		t.Errorf("Expected output did not appear: %v", err)
	}

	// Wait for command to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Run() did not complete in time")
	}
}

func TestRun_InvalidCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mockExec := mocks.NewMockExec()
	deps := &config.Dependencies{
		ExecCommand: mockExec.Command,
	}

	conn := newFakeConn()

	// Send an unsupported command to the mock shell
	conn.readBuf.WriteString("unsupported-command\n")
	conn.readBuf.WriteString("exit\n")

	// Run the command in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, conn, "/bin/sh", deps)
	}()

	// Wait for expected output to appear (Run() may return before pipeio.Pipe finishes copying)
	if err := conn.waitForOutput("command not supported", 2*time.Second); err != nil {
		t.Errorf("Expected output did not appear: %v", err)
	}

	// Wait for command to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Run() did not complete in time")
	}
}

func TestRun_Whoami(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mockExec := mocks.NewMockExec()
	deps := &config.Dependencies{
		ExecCommand: mockExec.Command,
	}

	conn := newFakeConn()

	// Prepare input for the mock shell
	conn.readBuf.WriteString("whoami\n")
	conn.readBuf.WriteString("exit\n")

	// Run the command in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, conn, "/bin/sh", deps)
	}()

	// Wait for expected output to appear (Run() may return before pipeio.Pipe finishes copying)
	if err := conn.waitForOutput("mockcmd[/bin/sh]", 2*time.Second); err != nil {
		t.Errorf("Expected output did not appear: %v", err)
	}

	// Wait for command to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Run() did not complete in time")
	}
}
