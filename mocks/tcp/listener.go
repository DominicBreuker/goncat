package tcp

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// MockTCPListener is a mock implementation of net.TCPListener.
type MockTCPListener struct {
	addr       *net.TCPAddr
	connCh     chan *MockTCPConn
	acceptedCh chan *MockTCPConn
	closeCh    chan struct{}
	closed     bool
	mu         sync.Mutex
	network    *MockTCPNetwork
}

// Accept waits for and returns the next connection to the listener.
func (l *MockTCPListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connCh:
		// Notify any waiter that the connection was accepted. Use a non-blocking send
		// so Accept behavior doesn't change if nobody is waiting.
		select {
		case l.acceptedCh <- conn:
		default:
		}

		return conn, nil
	case <-l.closeCh:
		return nil, fmt.Errorf("listener closed")
	}
}

// Close closes the listener.
func (l *MockTCPListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true
	close(l.closeCh)

	// Remove the listener from the network's map
	l.network.mu.Lock()
	delete(l.network.listeners, l.addr.String())
	l.network.mu.Unlock()

	return nil
}

// Addr returns the listener's network address.
func (l *MockTCPListener) Addr() net.Addr {
	return l.addr
}

var _ net.Listener = (*MockTCPListener)(nil)

// WaitForNewConnection waits for a new connection to arrive on the listener's
// incoming connection channel. It blocks until a connection is available,
// the listener is closed, or the timeout (in milliseconds) elapses. Returns
// the accepted *mockTCPConn on success.
func (l *MockTCPListener) WaitForNewConnection(timeoutMs int) (*MockTCPConn, error) {
	timeout := time.Duration(timeoutMs) * time.Millisecond

	select {
	// Wait for the connection that has been accepted by Accept()
	case conn := <-l.acceptedCh:
		return conn, nil
	case <-l.closeCh:
		return nil, fmt.Errorf("listener closed")
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for new connection on %s", l.addr.String())
	}
}
