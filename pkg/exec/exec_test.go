package exec

import (
	"bytes"
	"context"
	"io"
	"net"
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
	if testing.Short() {
		t.Skip("skipping exec test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := newFakeConn()

	// Run echo command with input
	conn.readBuf.WriteString("test input\n")

	go func() {
		// Give some time for the command to start and process
		time.Sleep(100 * time.Millisecond)
		cancel() // Cancel to stop the blocking Run
	}()

	// This will block until cancelled or command completes
	err := Run(ctx, conn, "echo")

	// We expect either no error or a context cancellation
	if err != nil && ctx.Err() == nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestRun_InvalidCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping exec test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn := newFakeConn()

	err := Run(ctx, conn, "nonexistent-command-12345")
	if err == nil {
		t.Error("Run() with invalid command should return error")
	}
}
