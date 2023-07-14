package socks

import (
	"bufio"
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
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
}

type Config struct {
	LocalHost string
	LocalPort int
}

func (cfg Config) String() string {
	return fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort)
}

// ServerControlSession ...
type ServerControlSession interface {
	SendAndGetOneChannel(m msg.Message) (net.Conn, error)
}

// NewServer ...
func NewServer(ctx context.Context, cfg Config, sessCtl ServerControlSession) *Server {
	return &Server{
		ctx:     ctx,
		cfg:     cfg,
		sessCtl: sessCtl,
	}
}

// Handle ...
func (srv *Server) Handle() error {
	addr := fmt.Sprintf("%s:%d", srv.cfg.LocalHost, srv.cfg.LocalPort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("listen(tcp, %s): %s", addr, err)
	}

	go func() {
		<-srv.ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if srv.ctx.Err() != nil {
				return nil // cancelled
			}

			log.ErrorMsg("SOCKS proxy: Accept(): %s\n", err)
			continue
		}

		go func() {
			defer conn.Close()

			if err := srv.handle(conn); err != nil {
				log.ErrorMsg("SOCKS proxy: handling connection: %s\n", err)
			}
		}()
	}
}

func (srv *Server) handle(connLocal net.Conn) error {
	buffered := bufio.NewReader(connLocal)

	msr, err := socks.ReadMethodSelectionRequest(buffered)
	if err != nil {
		return fmt.Errorf("reading method selection request: %s", err)
	}

	if !msr.IsNoAuthRequested() {
		if err := socks.WriteMethodSelectionResponse(connLocal, socks.MethodNoAcceptableMethods); err != nil {
			return fmt.Errorf("writing NoAcceptableMethods response: %s", err)
		}

		return fmt.Errorf("requested methods (%+v) did not include %d (NoAuthenticationRequired) but that is all we support", msr.Methods, socks.MethodNoAuthenticationRequired)
	}

	if err := socks.WriteMethodSelectionResponse(connLocal, socks.MethodNoAuthenticationRequired); err != nil {
		return fmt.Errorf("writing NoAuthenticationRequired response: %s", err)
	}

	sr, err := socks.ReadRequest(buffered)
	if err != nil {
		if errors.Is(err, socks.ErrCommandNotSupported) {
			if err := socks.WriteReplyError(connLocal, socks.ReplyCommandNotSupported); err != nil {
				return fmt.Errorf("writing Reply error response: %s", err)
			}

			return fmt.Errorf("reading SocksRequest: %s", err)
		}

		if errors.Is(err, socks.ErrAddressTypeNotSupported) {
			if err := socks.WriteReplyError(connLocal, socks.ReplyCommandNotSupported); err != nil {
				return fmt.Errorf("writing Reply error response: %s", err)
			}

			return fmt.Errorf("reading SocksRequest: %s", err)
		}

		if err := socks.WriteReplyError(connLocal, socks.ReplyGeneralFailure); err != nil {
			return fmt.Errorf("writing Reply error response: %s", err)
		}

		return fmt.Errorf("reading SocksRequest: %s", err)
	}

	m := msg.SocksConnect{
		RemoteHost: sr.DstAddr.String(),
		RemotePort: sr.DstPort,
	}

	connRemote, err := srv.sessCtl.SendAndGetOneChannel(m)
	if err != nil {
		return fmt.Errorf("SendAndGetOneChannel() for conn: %s", err)
	}
	defer connRemote.Close()

	pipeio.Pipe(srv.ctx, connLocal, connRemote, func(err error) {
		log.ErrorMsg("Pipe(stdio, conn): %s\n", err)
	})

	return nil
}
