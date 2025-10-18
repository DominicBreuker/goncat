package tcp

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"dominicbreuker/goncat/pkg/config"
)

// Server is a simple line-oriented echo server used for tests.
// For every line received it replies with the configured Prefix + the line.
// The server keeps accepted connections open until the remote side closes
// them or Close() is called. Close() will stop accepting new connections
// and close all active connections.
type Server struct {
	listener net.Listener
	prefix   string

	// protect access to conns
	mu    sync.Mutex
	conns map[net.Conn]struct{}

	// shutdown coordination
	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewServer creates a Server by calling the provided TCP listener function
// (see pkg/config). The listener func is expected to create and return a
// net.Listener (for example the project's mock network listener). The
// network and address string (like "127.0.0.1:9000" or ":0") are used to
// resolve a *net.TCPAddr that is forwarded to the listener function. The
// server will use the provided prefix for responses.
func NewServer(listener config.TCPListenerFunc, network string, addr string, prefix string) (*Server, error) {
	if listener == nil {
		return nil, fmt.Errorf("listener func is nil")
	}

	// Resolve address string into *net.TCPAddr for the listener func.
	laddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return nil, err
	}

	ln, err := listener(network, laddr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: ln,
		prefix:   prefix,
		conns:    make(map[net.Conn]struct{}),
		closed:   make(chan struct{}),
	}

	s.wg.Add(1)
	go s.acceptLoop()

	return s, nil
}

// Addr returns the actual listening address (useful when using :0).
func (s *Server) Addr() net.Addr {
	if s == nil || s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// Close stops the server from accepting new connections and closes all
// active connections. It waits for outstanding goroutines to finish and
// returns once shutdown is complete.
func (s *Server) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		// Stop accepting
		if s.listener != nil {
			err = s.listener.Close()
		}
		// Close all active conns
		s.mu.Lock()
		for c := range s.conns {
			// set a short deadline to unblock reads/writes
			_ = c.SetDeadline(time.Now().Add(50 * time.Millisecond))
			c.Close()
		}
		s.mu.Unlock()

		// Wait for accept loop and per-connection goroutines
		s.wg.Wait()
	})
	return err
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.closed:
				// Expected during shutdown
				return
			default:
				// transient error? wait a bit then continue
				time.Sleep(50 * time.Millisecond)
				// try again unless closed
				select {
				case <-s.closed:
					return
				default:
				}
				continue
			}
		}

		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(c net.Conn) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		delete(s.conns, c)
		s.mu.Unlock()
		c.Close()
	}()

	// Use a buffered scanner to read lines. Increase buffer if necessary.
	scanner := bufio.NewScanner(c)
	// allow reasonably long lines in tests
	const maxTokenSize = 64 * 1024
	buf := make([]byte, 4096)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Text()
		resp := fmt.Sprintf("%s%s\n", s.prefix, line)
		// Best-effort write; if the write fails, break and close conn
		_, err := c.Write([]byte(resp))
		if err != nil {
			return
		}

		// If server is closed, return to terminate connection
		select {
		case <-s.closed:
			return
		default:
		}
	}

	// scanner finished due to EOF or error; just return
}
