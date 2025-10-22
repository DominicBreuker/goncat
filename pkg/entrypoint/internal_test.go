package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

// runAsync runs a function in a goroutine and returns an error channel.
func runAsync(fn func() error) chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn()
	}()
	return errCh
}

// waitForError waits for an error from a channel with timeout.
func waitForError(t *testing.T, errCh chan error, timeout time.Duration) error {
	t.Helper()
	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		t.Fatal("timeout waiting for function to complete")
		return nil
	}
}

// assertNoError fails the test if err is not nil.
func assertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: got error %v, want nil", msg, err)
	}
}

// assertError fails the test if err is nil.
func assertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: got nil, want error", msg)
	}
}

// fakeServer implements a fake server for testing.
type fakeServer struct {
	serveErr  error
	serveCh   chan struct{}
	closed    bool
	closeCh   chan struct{}
	closeErr  error
	closeWait time.Duration
	mu        sync.Mutex
}

func (f *fakeServer) Serve() error {
	if f.serveCh != nil {
		<-f.serveCh
	}
	return f.serveErr
}

func (f *fakeServer) Close() error {
	if f.closeWait > 0 {
		time.Sleep(f.closeWait)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		if f.closeCh != nil {
			close(f.closeCh)
		}
	}
	return f.closeErr
}

// fakeClient implements clientInterface for testing.
type fakeClient struct {
	connectErr error
	closeErr   error
	closed     bool
	closeCh    chan struct{}
	conn       net.Conn
	mu         sync.Mutex
}

func (f *fakeClient) Connect() error {
	return f.connectErr
}

func (f *fakeClient) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		if f.closeCh != nil {
			close(f.closeCh)
		}
	}
	return f.closeErr
}

func (f *fakeClient) GetConnection() net.Conn {
	return f.conn
}

// fakeSlave implements a fake slave handler for testing.
type fakeSlave struct {
	handleErr  error
	handleFunc func() error
	closed     bool
	mu         sync.Mutex
}

func (f *fakeSlave) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeSlave) Handle() error {
	if f.handleFunc != nil {
		return f.handleFunc()
	}
	return f.handleErr
}

// testConfig creates a standard test configuration.
func testConfig() *config.Shared {
	return &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}
}

// fakeConn implements a fake net.Conn for testing.
type fakeConn struct {
	closed  bool
	closeCh chan struct{}
	mu      sync.Mutex
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *fakeConn) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		if f.closeCh != nil {
			close(f.closeCh)
		}
	}
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (f *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}
}

func (f *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (f *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (f *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// fakeServerWithCloseCount wraps fakeServer to count Close calls.
type fakeServerWithCloseCount struct {
	*fakeServer
	closeCount int
}

func (f *fakeServerWithCloseCount) Close() error {
	f.closeCount++
	return f.fakeServer.Close()
}

func newFakeMasterHandle(err error, cb func()) masterHandler {
	return func(ctx context.Context, cfg *config.Shared, mCfg *config.Master, conn net.Conn) error {
		if cb != nil {
			cb()
		}
		return err
	}
}

func newFakeSlaveHandle(err error, cb func()) slaveHandler {
	return func(context.Context, *config.Shared, net.Conn) error {
		if cb != nil {
			cb()
		}
		return err
	}
}
