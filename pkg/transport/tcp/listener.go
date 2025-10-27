package tcp

import (
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Listener implements the transport.Listener interface for TCP connections.
// It allows up to 100 concurrent connections to prevent resource exhaustion.
type Listener struct {
	nl  net.Listener
	sem chan struct{} // capacity 100 -> allows up to 100 concurrent handlers
}

// NewListener creates a new TCP listener on the specified address.
// The deps parameter is optional and can be nil to use default implementations.
func NewListener(addr string, deps *config.Dependencies) (*Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}

	listenerFn := config.GetTCPListenerFunc(deps)

	nl, err := listenerFn("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("listen(tcp, %s): %w", addr, err)
	}

	l := &Listener{
		nl:  nl,
		sem: make(chan struct{}, 100),
	}
	// initially allow 100 active connections
	for i := 0; i < 100; i++ {
		l.sem <- struct{}{}
	}
	return l, nil
}

// Serve accepts and handles incoming connections using the provided handler.
// Up to 100 connections can be handled concurrently; additional connections are closed.
func (l *Listener) Serve(handle transport.Handler) error {
	for {
		conn, err := l.nl.Accept()
		if err != nil {
			// Treat listener closed as clean shutdown.
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			// Retry on timeouts with a short backoff.
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("Accept(): %w", err)
		}

		// Try to acquire a slot. If all 100 slots busy, drop the new connection.
		select {
		case <-l.sem: // acquired
			go func(c net.Conn) {
				// Always release the slot and close the connection when done.
				defer func() {
					_ = c.Close()
					l.sem <- struct{}{} // release slot
				}()
				// Prevent a panic from leaking the slot.
				defer func() {
					if r := recover(); r != nil {
						log.ErrorMsg("Handler panic: %v\n", r)
					}
				}()

				if err := handle(c); err != nil {
					log.ErrorMsg("Handling connection: %s\n", err)
				}
			}(conn)

		default:
			// All 100 slots busy: politely close the extra connection.
			_ = conn.Close()
			// continue accepting
		}
	}
}

// Close stops the listener.
func (l *Listener) Close() error {
	return l.nl.Close()
}
