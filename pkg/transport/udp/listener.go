package udp

import (
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	kcp "github.com/xtaci/kcp-go/v5"
)

// Listener implements the transport.Listener interface for UDP connections with KCP.
// It ensures only one connection is handled at a time via a semaphore.
type Listener struct {
	kcpListener *kcp.Listener
	sem         chan struct{} // capacity 1 -> allows a single active handler
}

// NewListener creates a new UDP listener with KCP on the specified address.
// The deps parameter is optional and can be nil to use default implementations.
func NewListener(addr string, deps *config.Dependencies) (*Listener, error) {
	// Parse UDP address (not strictly necessary but good for validation)
	_, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveUDPAddr(udp, %s): %w", addr, err)
	}

	// Create UDP packet connection using stdlib
	packetConnFn := config.GetPacketListenerFunc(deps)
	conn, err := packetConnFn("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen(udp, %s): %w", addr, err)
	}

	// Create KCP listener from packet connection
	// ServeConn wraps a PacketConn and returns a KCP Listener
	// Parameters: block cipher (nil for no encryption), dataShards (0), parityShards (0), conn
	kcpListener, err := kcp.ServeConn(nil, 0, 0, conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("kcp.ServeConn(): %w", err)
	}

	l := &Listener{
		kcpListener: kcpListener,
		sem:         make(chan struct{}, 1),
	}
	// initially allow one active connection
	l.sem <- struct{}{}
	return l, nil
}

// Serve accepts and handles incoming KCP connections using the provided handler.
// Only one connection is handled at a time; additional connections are closed if received.
func (l *Listener) Serve(handle transport.Handler) error {
	for {
		// Accept KCP session (blocks until connection)
		kcpConn, err := l.kcpListener.AcceptKCP()
		if err != nil {
			// Treat listener closed as clean shutdown.
			if errors.Is(err, net.ErrClosed) || 
			   errors.Is(err, io.ErrClosedPipe) ||
			   strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			return fmt.Errorf("AcceptKCP(): %w", err)
		}

		// Configure KCP session
		kcpConn.SetNoDelay(1, 10, 2, 1)
		kcpConn.SetStreamMode(true)
		kcpConn.SetWindowSize(1024, 1024)

		// Try to acquire the single slot. If busy, drop the new connection.
		select {
		case <-l.sem: // acquired
			go func(c *kcp.UDPSession) {
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
			}(kcpConn)

		default:
			// Already handling one: politely close the extra connection.
			_ = kcpConn.Close()
			// continue accepting
		}
	}
}

// Close stops the listener.
func (l *Listener) Close() error {
	return l.kcpListener.Close()
}
