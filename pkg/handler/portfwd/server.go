// Package portfwd provides client and server implementations for port forwarding
// over multiplexed connections. It enables forwarding local ports to remote destinations
// and vice versa through the established connection.
package portfwd

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Server handles incoming connections on a local port and forwards them
// to a remote destination through a multiplexed control session.
type Server struct {
	ctx        context.Context
	cfg        Config
	sessCtl    ServerControlSession
	listenerFn config.TCPListenerFunc
}

// Config contains the configuration for port forwarding, specifying both
// the local endpoint to listen on and the remote destination to forward to.
type Config struct {
	LocalHost  string // Local host address to listen on
	LocalPort  int    // Local port to listen on
	RemoteHost string // Remote host to forward connections to
	RemotePort int    // Remote port to forward connections to
}

// ServerControlSession represents the interface for communicating over
// a multiplexed control session to establish new forwarding channels.
type ServerControlSession interface {
	SendAndGetOneChannelContext(ctx context.Context, m msg.Message) (net.Conn, error)
}

// String returns a human-readable string representation of the port forwarding configuration.
func (cfg Config) String() string {
	return fmt.Sprintf("PortForwarding[%s:%d -> %s:%d]", cfg.LocalHost, cfg.LocalPort, cfg.RemoteHost, cfg.RemotePort)
}

// NewServer creates a new port forwarding server with the given configuration.
// The deps parameter is optional and can be nil to use default implementations.
func NewServer(ctx context.Context, cfg Config, sessCtl ServerControlSession, deps *config.Dependencies) *Server {
	return &Server{
		ctx:        ctx,
		cfg:        cfg,
		sessCtl:    sessCtl,
		listenerFn: config.GetTCPListenerFunc(deps),
	}
}

// Handle starts listening on the configured local port and forwards accepted
// connections to the remote destination. It blocks until the context is cancelled
// or an unrecoverable error occurs.
func (srv *Server) Handle() error {
	addr := format.Addr(srv.cfg.LocalHost, srv.cfg.LocalPort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}

	l, err := srv.listenerFn("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("listen(tcp, %s): %w", addr, err)
	}

	go func() {
		<-srv.ctx.Done()
		l.Close()
	}()

	for {
		conn, err := srv.acceptWithContext(l)
		if err != nil {
			if srv.ctx.Err() != nil {
				return nil // cancelled
			}

			// If the listener is closed, exit cleanly.
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}

			log.ErrorMsg("Port forwarding %s: Accept(): %w", srv.cfg, err)
			time.Sleep(100 * time.Millisecond) // tiny backoff to avoid a tight loop
			continue
		}

		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetKeepAlive(true)
		}

		go func() {
			defer func() {
				_ = conn.Close()
				if r := recover(); r != nil {
					log.ErrorMsg("Port forwarding %s: handler panic: %v\n", srv.cfg, r)
				}
			}()
			if err := srv.handlePortForwardingConn(conn); err != nil {
				log.ErrorMsg("Port forwarding %s: handling connection: %s\n", srv.cfg, err)
			}
		}()
	}
}

// acceptWithContext accepts from the provided listener but returns early when
// srv.ctx is cancelled. It runs Accept() in a goroutine and selects on the
// server context to avoid leaking goroutines when the caller cancels.
func (srv *Server) acceptWithContext(l net.Listener) (net.Conn, error) {
	type res struct {
		c   net.Conn
		err error
	}

	ch := make(chan res, 1)

	go func() {
		c, e := l.Accept()
		// Do not block if the caller already returned due to ctx.Done().
		select {
		case ch <- res{c: c, err: e}:
		case <-srv.ctx.Done():
			if c != nil {
				_ = c.Close()
			}
		}
	}()

	select {
	case <-srv.ctx.Done():
		return nil, srv.ctx.Err()
	case r := <-ch:
		return r.c, r.err
	}
}

func (srv *Server) handlePortForwardingConn(connLocal net.Conn) error {
	m := msg.Connect{
		RemoteHost: srv.cfg.RemoteHost,
		RemotePort: srv.cfg.RemotePort,
	}

	connRemote, err := srv.sessCtl.SendAndGetOneChannelContext(srv.ctx, m)
	if err != nil {
		return fmt.Errorf("SendAndGetOneChannel() for conn: %w", err)
	}
	defer connRemote.Close()

	pipeio.Pipe(srv.ctx, connLocal, connRemote, func(err error) {
		log.ErrorMsg("port forwarding %s: pipe error: %s\n", srv.cfg, err)
	})

	return nil
}
