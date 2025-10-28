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
	"sync"
	"time"
)

// Server handles incoming connections on a local port and forwards them
// to a remote destination through a multiplexed control session.
type Server struct {
	ctx           context.Context
	cfg           Config
	sessCtl       ServerControlSession
	tcpListenerFn config.TCPListenerFunc
	udpListenerFn config.UDPListenerFunc
}

// Config contains the configuration for port forwarding, specifying both
// the local endpoint to listen on and the remote destination to forward to.
type Config struct {
	Protocol   string        // "tcp" or "udp"
	LocalHost  string        // Local host address to listen on
	LocalPort  int           // Local port to listen on
	RemoteHost string        // Remote host to forward connections to
	RemotePort int           // Remote port to forward connections to
	Timeout    time.Duration // Timeout for operations (used for UDP session cleanup)
	Logger     *log.Logger   // Logger for verbose messages
}

// ServerControlSession represents the interface for communicating over
// a multiplexed control session to establish new forwarding channels.
type ServerControlSession interface {
	SendAndGetOneChannelContext(ctx context.Context, m msg.Message) (net.Conn, error)
}

// String returns a human-readable string representation of the port forwarding configuration.
func (cfg Config) String() string {
	protocol := cfg.Protocol
	if protocol == "" {
		protocol = "tcp"
	}
	return fmt.Sprintf("PortForwarding[%s:%s:%d -> %s:%d]", protocol, cfg.LocalHost, cfg.LocalPort, cfg.RemoteHost, cfg.RemotePort)
}

// NewServer creates a new port forwarding server with the given configuration.
// The deps parameter is optional and can be nil to use default implementations.
func NewServer(ctx context.Context, cfg Config, sessCtl ServerControlSession, deps *config.Dependencies) *Server {
	return &Server{
		ctx:           ctx,
		cfg:           cfg,
		sessCtl:       sessCtl,
		tcpListenerFn: config.GetTCPListenerFunc(deps),
		udpListenerFn: config.GetUDPListenerFunc(deps),
	}
}

// Handle starts listening on the configured local port and forwards accepted
// connections to the remote destination. It blocks until the context is cancelled
// or an unrecoverable error occurs.
func (srv *Server) Handle() error {
	protocol := srv.cfg.Protocol
	if protocol == "" {
		protocol = "tcp" // default to TCP for backward compatibility
	}

	switch protocol {
	case "tcp":
		return srv.handleTCP()
	case "udp":
		return srv.handleUDP()
	default:
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// handleTCP handles TCP port forwarding by accepting TCP connections
// and forwarding them through yamux streams.
func (srv *Server) handleTCP() error {
	addr := format.Addr(srv.cfg.LocalHost, srv.cfg.LocalPort)
	remoteAddr := format.Addr(srv.cfg.RemoteHost, srv.cfg.RemotePort)

	srv.cfg.Logger.VerboseMsg("Port forwarding: listening on %s (forwarding to %s)", addr, remoteAddr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}

	l, err := srv.tcpListenerFn("tcp", tcpAddr)
	if err != nil {
		srv.cfg.Logger.VerboseMsg("Port forwarding error: failed to listen on %s: %v", addr, err)
		return fmt.Errorf("listen(tcp, %s): %w", addr, err)
	}

	go func() {
		<-srv.ctx.Done()
		srv.cfg.Logger.VerboseMsg("Port forwarding: context cancelled, closing listener on %s", addr)
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

			srv.cfg.Logger.VerboseMsg("Port forwarding error: Accept() on %s: %v", addr, err)
			srv.cfg.Logger.ErrorMsg("Port forwarding %s: Accept(): %w", srv.cfg, err)
			time.Sleep(100 * time.Millisecond) // tiny backoff to avoid a tight loop
			continue
		}

		srv.cfg.Logger.VerboseMsg("Port forwarding: accepted connection from %s", conn.RemoteAddr())

		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetKeepAlive(true)
		}

		go func() {
			defer func() {
				srv.cfg.Logger.VerboseMsg("Port forwarding: connection from %s closed", conn.RemoteAddr())
				_ = conn.Close()
				if r := recover(); r != nil {
					srv.cfg.Logger.ErrorMsg("Port forwarding %s: handler panic: %v\n", srv.cfg, r)
				}
			}()
			if err := srv.handleTCPConn(conn); err != nil {
				srv.cfg.Logger.ErrorMsg("Port forwarding %s: handling connection: %s\n", srv.cfg, err)
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

func (srv *Server) handleTCPConn(connLocal net.Conn) error {
	remoteAddr := format.Addr(srv.cfg.RemoteHost, srv.cfg.RemotePort)
	srv.cfg.Logger.VerboseMsg("Port forwarding: creating forwarding stream for %s to %s", connLocal.RemoteAddr(), remoteAddr)

	m := msg.Connect{
		Protocol:   "tcp",
		RemoteHost: srv.cfg.RemoteHost,
		RemotePort: srv.cfg.RemotePort,
	}

	connRemote, err := srv.sessCtl.SendAndGetOneChannelContext(srv.ctx, m)
	if err != nil {
		srv.cfg.Logger.VerboseMsg("Port forwarding error: failed to create stream for %s: %v", connLocal.RemoteAddr(), err)
		return fmt.Errorf("SendAndGetOneChannel() for conn: %w", err)
	}
	defer connRemote.Close()

	srv.cfg.Logger.VerboseMsg("Port forwarding: piping data for %s", connLocal.RemoteAddr())
	pipeio.Pipe(srv.ctx, connLocal, connRemote, func(err error) {
		srv.cfg.Logger.ErrorMsg("port forwarding %s: pipe error: %s\n", srv.cfg, err)
	})

	return nil
}

// handleUDP handles UDP port forwarding by receiving datagrams on a UDP socket
// and forwarding them through yamux streams. It maintains a map of client addresses
// to yamux streams to route responses back to the correct client.
func (srv *Server) handleUDP() error {
	addr := format.Addr(srv.cfg.LocalHost, srv.cfg.LocalPort)
	remoteAddr := format.Addr(srv.cfg.RemoteHost, srv.cfg.RemotePort)

	srv.cfg.Logger.VerboseMsg("Port forwarding UDP: listening on %s (forwarding to %s)", addr, remoteAddr)

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveUDPAddr(udp, %s): %w", addr, err)
	}

	conn, err := srv.udpListenerFn("udp", udpAddr)
	if err != nil {
		srv.cfg.Logger.VerboseMsg("Port forwarding UDP error: failed to listen on %s: %v", addr, err)
		return fmt.Errorf("listen(udp, %s): %w", addr, err)
	}

	// Set a read deadline for debugging

	go func() {
		<-srv.ctx.Done()
		srv.cfg.Logger.VerboseMsg("Port forwarding UDP: context cancelled, closing listener on %s", addr)
		conn.Close()
	}()

	// Map of client address -> UDP session info
	type udpSession struct {
		stream     net.Conn
		lastActive time.Time
		cancel     context.CancelFunc
	}
	sessions := make(map[string]*udpSession)
	var sessionsMu sync.Mutex

	// Cleanup idle sessions periodically
	timeout := srv.cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second // Default UDP session timeout if not configured
	}
	go func() {
		ticker := time.NewTicker(timeout / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sessionsMu.Lock()
				now := time.Now()
				for addr, sess := range sessions {
					if now.Sub(sess.lastActive) > timeout {
						srv.cfg.Logger.VerboseMsg("Port forwarding UDP: cleaned up session for %s (idle timeout)", addr)
						sess.cancel()
						sess.stream.Close()
						delete(sessions, addr)
					}
				}
				sessionsMu.Unlock()
			case <-srv.ctx.Done():
				return
			}
		}
	}()

	buffer := make([]byte, 65536) // Max UDP datagram size
	for {
		// Reset read deadline for each iteration

		n, clientAddr, err := conn.ReadFrom(buffer)
		if err != nil {
			if srv.ctx.Err() != nil {
				return nil // cancelled
			}
			if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			srv.cfg.Logger.ErrorMsg("UDP port forwarding %s: ReadFrom(): %s\n", srv.cfg, err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		data := make([]byte, n)
		copy(data, buffer[:n])

		// Handle this datagram in a goroutine to avoid blocking
		go func(clientAddr net.Addr, data []byte) {
			sessionKey := clientAddr.String()

			sessionsMu.Lock()
			sess, exists := sessions[sessionKey]
			if !exists {
				srv.cfg.Logger.VerboseMsg("Port forwarding UDP: creating session for %s", clientAddr)
				// Create new session for this client
				ctx, cancel := context.WithCancel(srv.ctx)
				sess = &udpSession{
					lastActive: time.Now(),
					cancel:     cancel,
				}

				// Open yamux stream for this client
				m := msg.Connect{
					Protocol:   "udp",
					RemoteHost: srv.cfg.RemoteHost,
					RemotePort: srv.cfg.RemotePort,
				}

				stream, err := srv.sessCtl.SendAndGetOneChannelContext(ctx, m)
				if err != nil {
					sessionsMu.Unlock()
					cancel()
					srv.cfg.Logger.VerboseMsg("Port forwarding UDP error: failed to open stream for %s: %v", clientAddr, err)
					srv.cfg.Logger.ErrorMsg("UDP port forwarding %s: failed to open stream for %s: %s\n", srv.cfg, clientAddr, err)
					return
				}

				sess.stream = stream
				sessions[sessionKey] = sess

				// Start goroutine to read responses and send back to client
				go func() {
					defer func() {
						sessionsMu.Lock()
						delete(sessions, sessionKey)
						sessionsMu.Unlock()
						stream.Close()
						cancel()
					}()

					respBuffer := make([]byte, 65536)
					for {
						n, err := stream.Read(respBuffer)
						if err != nil {
							if ctx.Err() != nil {
								return // context cancelled
							}
							return
						}

						// Send response back to client
						_, err = conn.WriteTo(respBuffer[:n], clientAddr)
						if err != nil {
							srv.cfg.Logger.ErrorMsg("UDP port forwarding %s: WriteTo(%s): %s\n", srv.cfg, clientAddr, err)
							return
						}

						// Update last active time
						sessionsMu.Lock()
						if s, ok := sessions[sessionKey]; ok {
							s.lastActive = time.Now()
						}
						sessionsMu.Unlock()
					}
				}()
			}
			sess.lastActive = time.Now()
			stream := sess.stream
			sessionsMu.Unlock()

			// Forward datagram to remote through yamux stream
			_, err := stream.Write(data)
			if err != nil {
				srv.cfg.Logger.ErrorMsg("UDP port forwarding %s: Write to stream for %s: %s\n", srv.cfg, clientAddr, err)
				sessionsMu.Lock()
				if s, ok := sessions[sessionKey]; ok {
					s.cancel()
					s.stream.Close()
					delete(sessions, sessionKey)
				}
				sessionsMu.Unlock()
			}
		}(clientAddr, data)
	}
}
