package tcp

import (
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Listener implements the transport.Listener interface for TCP connections.
// It ensures only one connection is handled at a time.
type Listener struct {
	nl net.Listener

	rdy bool // whether we can handle a new connection
	mu  sync.Mutex
}

// NewListener creates a new TCP listener on the specified address.
// The deps parameter is optional and can be nil to use default implementations.
func NewListener(addr string, deps *config.Dependencies) (*Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	listenerFn := config.GetTCPListenerFunc(deps)

	nl, err := listenerFn("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("listen(tcp, %s): %s", addr, err)
	}

	return &Listener{
		nl:  nl,
		rdy: true,
	}, nil
}

// Serve accepts and handles incoming connections using the provided handler.
// Only one connection is handled at a time; additional connections are closed if received.
func (l *Listener) Serve(handle transport.Handler) error {
	for {
		conn, err := l.nl.Accept()
		if err != nil {
			// Treat listener closed as clean shutdown.
			// Prefer net.ErrClosed detection where possible, but also tolerate
			// implementations that return the textual "use of closed network connection".
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}

			// Retry on timeouts with a short backoff. net.Error.Temporary is deprecated,
			// prefer checking Timeout() to detect deadline/timeout conditions.
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			return fmt.Errorf("Accept(): %s", err)
		}

		// close if we are not ready

		l.mu.Lock()
		if !l.rdy {
			conn.Close() // we already handle a connection
			l.mu.Unlock()
			continue
		}
		l.rdy = false
		l.mu.Unlock()

		go func(c net.Conn) {
			// get ready again eventually
			defer func() {
				l.mu.Lock()
				l.rdy = true
				l.mu.Unlock()
			}()

			log.InfoMsg("New TCP connection from %s\n", c.RemoteAddr())

			if err := handle(c); err != nil {
				log.ErrorMsg("Handling connection: %s\n", err)
			}
		}(conn)
	}
}

// Close stops the listener.
func (l *Listener) Close() error {
	return l.nl.Close()
}
