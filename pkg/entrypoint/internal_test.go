package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"errors"
	"net"
	"sync"
	"time"
)

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
