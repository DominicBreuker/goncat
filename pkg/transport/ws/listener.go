package ws

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// Listener implements the transport.Listener interface for WebSocket connections.
// It wraps an HTTP server that upgrades incoming requests to WebSocket connections.
// Up to 100 connections can be handled concurrently; additional connections receive HTTP 503.
type Listener struct {
	ctx context.Context
	nl  net.Listener

	// semaphore (cap=100) to allow up to 100 concurrent connections
	sem chan struct{}
}

// NewListener creates a new WebSocket listener on the specified address.
// If useTLS is true, the listener will use TLS with an ephemeral self-signed certificate.
func NewListener(ctx context.Context, addr string, useTLS bool) (*Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}

	// Use the generic net.Listener interface (not *net.TCPListener)
	var nl net.Listener
	nl, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.ListenTCP(tcp, %s): %w", tcpAddr.String(), err)
	}

	if useTLS {
		nl, err = getTLSListener(nl)
		if err != nil {
			return nil, fmt.Errorf("getTLSListener(): %w", err)
		}
	}

	l := &Listener{
		ctx: ctx,
		nl:  nl,
		sem: make(chan struct{}, 100),
	}
	// initially allow 100 active connections
	for i := 0; i < 100; i++ {
		l.sem <- struct{}{}
	}
	return l, nil
}

func getTLSListener(nl net.Listener) (net.Listener, error) {
	// Ephemeral cert per process; clients skip-verify anyway.
	// (If you need mTLS/pinning, you’re already doing TLS-in-TLS at the app layer.)
	key := rand.Text()
	_, cert, err := crypto.GenerateCertificates(key)
	if err != nil {
		return nil, fmt.Errorf("crypto.GenerateCertificates(%s): %w", key, err)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}
	return tls.NewListener(nl, tlsCfg), nil
}

// Serve starts the HTTP server and handles incoming WebSocket upgrade requests.
func (l *Listener) Serve(handle transport.Handler) error {
	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to acquire a slot. If all 100 slots busy, reject.
			select {
			case <-l.sem:
				// release on return
				defer func() { l.sem <- struct{}{} }()

				c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
					Subprotocols: []string{"bin"},
				})
				if err != nil {
					log.ErrorMsg("websocket.Accept(): %s\n", err)
					// If upgrade fails, sem is already released by defer; nothing else to do.
					return
				}

				conn := websocket.NetConn(l.ctx, c, websocket.MessageBinary)
				log.InfoMsg("New WS connection from %s\n", conn.RemoteAddr())

				// Make sure the connection is always closed, and don’t leak the handler slot on panic.
				defer func() { _ = conn.Close() }()
				defer func() {
					if r := recover(); r != nil {
						log.ErrorMsg("Handler panic: %v\n", r)
					}
				}()

				if err := handle(conn); err != nil {
					log.ErrorMsg("handle websocket.NetConn: %s\n", err)
				}

			default:
				// All 100 slots busy: reject extra connections politely
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
		}),

		// For long-lived tunnels, keep these generous; avoid spuriously killing idle conns.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0,                // unlimited reads after headers
		WriteTimeout:      0,                // don't kill slow writers; app layer handles deadlines if needed
		IdleTimeout:       60 * time.Second, // typical default
	}

	// Serve will unblock when the underlying net.Listener is closed via Close().
	if err := s.Serve(l.nl); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http.Server.Serve(): %w", err)
	}
	return nil
}

// Close stops the listener (this unblocks Serve()).
func (l *Listener) Close() error {
	return l.nl.Close()
}
