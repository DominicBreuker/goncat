package portfwd

import (
	"context"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"fmt"
	"net"
)

// Server ...
type Server struct {
	ctx     context.Context
	cfg     Config
	sessCtl ServerControlSession
}

// Config ...
type Config struct {
	LocalHost string
	LocalPort int

	RemoteHost string
	RemotePort int
}

// ServerControlSession ...
type ServerControlSession interface {
	SendAndGetOneChannel(m msg.Message) (net.Conn, error)
}

func (cfg Config) String() string {
	return fmt.Sprintf("PortForwarding[%s:%d -> %s:%d]", cfg.LocalHost, cfg.LocalPort, cfg.RemoteHost, cfg.RemotePort)
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
	addr := format.Addr(srv.cfg.LocalHost, srv.cfg.LocalPort)

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

			log.ErrorMsg("Port forwarding %s: Accept(): %s\n", srv.cfg, err)
			continue
		}

		go func() {
			defer conn.Close()

			if err := srv.handlePortForwardingConn(conn); err != nil {
				log.ErrorMsg("Port forwarding %s: handling connection: %s", srv.cfg, err)
			}
		}()
	}
}

func (srv *Server) handlePortForwardingConn(connLocal net.Conn) error {
	m := msg.Connect{
		RemoteHost: srv.cfg.RemoteHost,
		RemotePort: srv.cfg.RemotePort,
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
