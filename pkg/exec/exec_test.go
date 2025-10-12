package exec

import (
	"bytes"
	"context"
	"dominicbreuker/goncat/mocks"
	"dominicbreuker/goncat/pkg/config"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeConn is a minimal fake implementation of net.Conn for testing.
type fakeConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (f *fakeConn) Read(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, io.EOF
	}
	return f.readBuf.Read(p)
}

func (f *fakeConn) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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

	// Run the command - it now waits for both cmd.Wait() and pipeio.Pipe() to complete
	err := Run(ctx, conn, "/bin/sh", deps)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify output - Run() now ensures all data is copied before returning
	conn.mu.Lock()
	output := conn.writeBuf.String()
	conn.mu.Unlock()
	if !strings.Contains(output, "hello world") {
		t.Errorf("Run() output = %q, want to contain 'hello world'", output)
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

	// Run the command - it now waits for both cmd.Wait() and pipeio.Pipe() to complete
	err := Run(ctx, conn, "/bin/sh", deps)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify that error message is in output - Run() now ensures all data is copied before returning
	conn.mu.Lock()
	output := conn.writeBuf.String()
	conn.mu.Unlock()
	if !strings.Contains(output, "command not supported") {
		t.Errorf("Run() output = %q, want to contain 'command not supported'", output)
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

	// Run the command - it now waits for both cmd.Wait() and pipeio.Pipe() to complete
	err := Run(ctx, conn, "/bin/sh", deps)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify output contains mock shell identifier - Run() now ensures all data is copied before returning
	conn.mu.Lock()
	output := conn.writeBuf.String()
	conn.mu.Unlock()
	if !strings.Contains(output, "mockcmd[/bin/sh]") {
		t.Errorf("Run() output = %q, want to contain 'mockcmd[/bin/sh]'", output)
	}
}
