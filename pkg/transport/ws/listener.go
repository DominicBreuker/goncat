package ws

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Listener implements the transport.Listener interface for WebSocket connections.
// It wraps an HTTP server that upgrades incoming requests to WebSocket connections.
// Only one connection is handled at a time; additional connections receive HTTP 500 errors.
type Listener struct {
	ctx context.Context

	nl net.Listener

	rdy bool
	mu  sync.Mutex
}

// NewListener creates a new WebSocket listener on the specified address.
// If tls is true, the listener will use TLS with an ephemeral self-signed certificate.
func NewListener(ctx context.Context, addr string, tls bool) (*Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	var nl net.Listener
	nl, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.ListenTCP(tcp, %s): %s", tcpAddr.String(), err)
	}

	if tls {
		nl, err = getTLSListener(nl)
		if err != nil {
			return nil, fmt.Errorf("getTLSListener(): %s", err)
		}
	}

	return &Listener{
		ctx: ctx,
		nl:  nl,
		rdy: true,
	}, nil
}

func getTLSListener(nl net.Listener) (net.Listener, error) {
	key := rand.Text() // new random certificate on each launch, client will ignore anyways

	_, cert, err := crypto.GenerateCertificates(key)
	if err != nil {
		return nil, fmt.Errorf("crypto.GenerateCertificates(%s): %s", key, err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	return tls.NewListener(nl, tlsCfg), nil
}

// Serve starts the HTTP server and handles incoming WebSocket upgrade requests.
// Each connection is passed to the provided handler after the WebSocket upgrade completes.
func (l *Listener) Serve(handle transport.Handler) error {
	s := &http.Server{
		Handler: newHandler(handle, l),

		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
	}

	err := s.Serve(l.nl)
	if err != nil {
		return fmt.Errorf("s.Serve(): %s", err)
	}

	return nil
}

type handler struct {
	handle transport.Handler
	l      *Listener
}

func newHandler(handle transport.Handler, l *Listener) handler {
	return handler{
		handle: handle,
		l:      l,
	}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// reject with 500 unless we are ready

	h.l.mu.Lock()
	if !h.l.rdy {
		w.WriteHeader(500)
		h.l.mu.Unlock()
		return
	}
	h.l.rdy = false
	h.l.mu.Unlock()

	// get ready again eventually

	defer func() {
		h.l.mu.Lock()
		h.l.rdy = true
		h.l.mu.Unlock()
	}()

	// now handle the request

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"bin"},
	})
	if err != nil {
		log.ErrorMsg("websocket.Accept(): %s\n", err)
		return
	}

	conn := websocket.NetConn(h.l.ctx, c, websocket.MessageBinary)
	log.InfoMsg("New WS connection from %s\n", conn.RemoteAddr())

	if err := h.handle(conn); err != nil {
		log.ErrorMsg("handle websocket.NetConn: %s\n", err)
		return
	}
}

// Close stops the listener.
func (l *Listener) Close() error {
	return l.nl.Close()
}
