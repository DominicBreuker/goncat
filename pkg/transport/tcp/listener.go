package tcp

import (
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"fmt"
	"net"
	"sync"
)

// Listener implements the transport.Listener interface for TCP connections.
// It ensures only one connection is handled at a time.
type Listener struct {
	nl net.Listener

	rdy bool // whether we can handle a new connection
	mu  sync.Mutex
}

// NewListener creates a new TCP listener on the specified address.
func NewListener(addr string) (*Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	nl, err := net.ListenTCP("tcp", tcpAddr)
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

		go func() {
			// get ready again eventually
			defer func() {
				l.mu.Lock()
				l.rdy = true
				l.mu.Unlock()
			}()

			log.InfoMsg("New TCP connection from %s\n", conn.RemoteAddr())

			err = handle(conn)
			if err != nil {
				log.ErrorMsg("Handling connection: %s\n", err)
			}
		}()
	}
}

// Close stops the listener.
func (l *Listener) Close() error {
	return l.nl.Close()
}
