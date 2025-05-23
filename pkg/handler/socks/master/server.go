package master

import (
	"bufio"
	"context"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/socks"
	"errors"
	"fmt"
	"net"
)

// Server ...
type Server struct {
	ctx     context.Context
	cfg     Config
	sessCtl ServerControlSession

	listener *net.TCPListener
}

// NewServer ...
func NewServer(ctx context.Context, cfg Config, sessCtl ServerControlSession) (*Server, error) {
	addr := format.Addr(cfg.LocalHost, cfg.LocalPort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("listen(tcp, %s): %s", addr, err)
	}

	go func() {
		<-ctx.Done()
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
	log.ErrorMsg("SOCKS proxy: "+format, a...)
}

// Serve ...
func (srv *Server) Serve() error {
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			if srv.ctx.Err() != nil {
				return nil // cancelled
			}

			srv.LogError("Accept(): %s\n", err)
			continue
		}

		go func() {
			defer conn.Close()

			if err := srv.handle(conn); err != nil {
				srv.LogError("handling connection: %s\n", err)
			}
		}()
	}
}

func (srv *Server) handle(connLocal net.Conn) error {
	bufConnLocal := bufio.NewReadWriter(bufio.NewReader(connLocal), bufio.NewWriter(connLocal))
	defer bufConnLocal.Flush()

	if err := handleMethodSelection(bufConnLocal); err != nil {
		return fmt.Errorf("handling method selection: %s", err)
	}
	if err := bufConnLocal.Flush(); err != nil {
		return fmt.Errorf("flushing method selection: %s", err)
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
		return srv.handleConnect(connLocal, req)
	case socks.CommandAssociate:
		return srv.handleAssociate(bufConnLocal, req)
	default:
		return fmt.Errorf("unexpected SOCKS command %v: this is a bug", req.Cmd)
	}

}

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
