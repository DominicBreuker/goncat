package master

import (
	"bufio"
	"context"
	"time"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/socks"
	"errors"
	"fmt"
	"net"
)

// Server implements a SOCKS5 proxy server on the master side.
// It accepts SOCKS5 client connections and forwards them through
// the control session to the slave.
type Server struct {
	ctx     context.Context
	cfg     Config
	sessCtl ServerControlSession

	listener net.Listener
}

// NewServer creates a new SOCKS5 proxy server that listens on the configured address.
func NewServer(ctx context.Context, cfg Config, sessCtl ServerControlSession) (*Server, error) {
	addr := format.Addr(cfg.LocalHost, cfg.LocalPort)

	cfg.Logger.VerboseMsg("SOCKS proxy: listening on %s", addr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	// Get the TCP listener function from dependencies or use default
	listenerFn := config.GetTCPListenerFunc(cfg.Deps)
	l, err := listenerFn("tcp", tcpAddr)
	if err != nil {
		cfg.Logger.VerboseMsg("SOCKS proxy error: failed to listen on %s: %v", addr, err)
		return nil, fmt.Errorf("listen(tcp, %s): %s", addr, err)
	}

	go func() {
		<-ctx.Done()
		cfg.Logger.VerboseMsg("SOCKS proxy: context cancelled, closing listener on %s", addr)
		l.Close()
	}()

	return &Server{
		ctx:      ctx,
		cfg:      cfg,
		sessCtl:  sessCtl,
		listener: l,
	}, nil
}

// LogError logs an error message with a prefix to indicate where it comes from
func (srv *Server) LogError(format string, a ...interface{}) {
	srv.cfg.Logger.ErrorMsg("SOCKS proxy: "+format, a...)
}

// Serve starts accepting SOCKS5 client connections and handles them.
// It blocks until the context is cancelled or an unrecoverable error occurs.
func (srv *Server) Serve() error {
	for {
		conn, err := srv.acceptWithContext()
		if err != nil {
			// If the server context was cancelled, exit cleanly.
			if srv.ctx.Err() != nil {
				return nil
			}

			srv.cfg.Logger.VerboseMsg("SOCKS proxy error: Accept(): %v", err)
			srv.LogError("Accept(): %s\n", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		srv.cfg.Logger.VerboseMsg("SOCKS proxy: accepted client connection from %s", conn.RemoteAddr())

		go func() {
			defer func() {
				srv.cfg.Logger.VerboseMsg("SOCKS proxy: connection from %s closed", conn.RemoteAddr())
				conn.Close()
			}()

			if err := srv.handle(conn); err != nil {
				srv.LogError("handling connection: %s\n", err)
			}
		}()
	}
}

// acceptWithContext accepts a connection from the server listener but returns
// early if the server context is cancelled. It uses a goroutine and a
// buffered result channel to avoid leaking the goroutine when ctx is done.
func (srv *Server) acceptWithContext() (net.Conn, error) {
	type res struct {
		c   net.Conn
		err error
	}

	ch := make(chan res, 1)
	go func() {
		c, e := srv.listener.Accept()
		ch <- res{c: c, err: e}
	}()

	select {
	case <-srv.ctx.Done():
		return nil, srv.ctx.Err()
	case r := <-ch:
		return r.c, r.err
	}
}

// handle processes a single SOCKS5 client connection through the complete
// SOCKS5 handshake and request handling flow.
func (srv *Server) handle(connLocal net.Conn) error {
	bufConnLocal := bufio.NewReadWriter(bufio.NewReader(connLocal), bufio.NewWriter(connLocal))
	defer bufConnLocal.Flush()
	// Bound the method selection phase so a misbehaving client cannot block
	// the handler indefinitely.
	if c, ok := connLocal.(interface{ SetReadDeadline(time.Time) error }); ok {
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
		defer c.SetReadDeadline(time.Time{})
	}

	srv.cfg.Logger.VerboseMsg("SOCKS proxy: negotiating method with %s", connLocal.RemoteAddr())
	if err := handleMethodSelection(bufConnLocal); err != nil {
		return fmt.Errorf("handling method selection: %s", err)
	}
	if err := bufConnLocal.Flush(); err != nil {
		return fmt.Errorf("flushing method selection: %s", err)
	}

	// Clear any previous deadline and set a new deadline for the request
	// parsing phase.
	if c, ok := connLocal.(interface{ SetReadDeadline(time.Time) error }); ok {
		_ = c.SetReadDeadline(time.Time{})
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
		defer c.SetReadDeadline(time.Time{})
	}

	req, err := handleRequest(bufConnLocal)
	if err != nil {
		return fmt.Errorf("handling request: %s", err)
	}
	if err := bufConnLocal.Flush(); err != nil {
		return fmt.Errorf("flushing request: %s", err)
	}

	switch req.Cmd {
	case socks.CommandConnect:
		srv.cfg.Logger.VerboseMsg("SOCKS proxy: CONNECT request from %s to %s:%d", connLocal.RemoteAddr(), req.DstAddr, req.DstPort)
		return srv.handleConnect(connLocal, req)
	case socks.CommandAssociate:
		srv.cfg.Logger.VerboseMsg("SOCKS proxy: UDP ASSOCIATE request from %s", connLocal.RemoteAddr())
		return srv.handleAssociate(bufConnLocal, req)
	default:
		return fmt.Errorf("unexpected SOCKS command %v: this is a bug", req.Cmd)
	}

}

// handleMethodSelection processes the SOCKS5 method selection phase.
// It only accepts connections requesting no authentication.
func handleMethodSelection(bufConnLocal *bufio.ReadWriter) error {
	msr, err := socks.ReadMethodSelectionRequest(bufConnLocal)
	if err != nil {
		return fmt.Errorf("reading method selection request: %s", err)
	}

	if !msr.IsNoAuthRequested() {
		if err := socks.WriteMethodSelectionResponse(bufConnLocal, socks.MethodNoAcceptableMethods); err != nil {
			return fmt.Errorf("writing NoAcceptableMethods response: %s", err)
		}

		return fmt.Errorf("requested methods (%+v) did not include %d (NoAuthenticationRequired) but that is all we support", msr.Methods, socks.MethodNoAuthenticationRequired)
	}

	if err := socks.WriteMethodSelectionResponse(bufConnLocal, socks.MethodNoAuthenticationRequired); err != nil {
		return fmt.Errorf("writing NoAuthenticationRequired response: %s", err)
	}

	return nil
}

// handleRequest reads and validates the SOCKS5 request from the client.
// It returns the parsed request or writes an appropriate error response.
func handleRequest(bufConnLocal *bufio.ReadWriter) (*socks.Request, error) {
	sr, err := socks.ReadRequest(bufConnLocal)
	if err != nil {
		if errors.Is(err, socks.ErrCommandNotSupported) {
			if err := socks.WriteReplyError(bufConnLocal, socks.ReplyCommandNotSupported); err != nil {
				return nil, fmt.Errorf("writing Reply error response: %s", err)
			}

			return nil, fmt.Errorf("reading SocksRequest: %s", err)
		}

		if errors.Is(err, socks.ErrAddressTypeNotSupported) {
			if err := socks.WriteReplyError(bufConnLocal, socks.ReplyCommandNotSupported); err != nil {
				return nil, fmt.Errorf("writing Reply error response: %s", err)
			}

			return nil, fmt.Errorf("reading SocksRequest: %s", err)
		}

		if err := socks.WriteReplyError(bufConnLocal, socks.ReplyGeneralFailure); err != nil {
			return nil, fmt.Errorf("writing Reply error response: %s", err)
		}

		return nil, fmt.Errorf("reading SocksRequest: %s", err)
	}

	return sr, nil
}
