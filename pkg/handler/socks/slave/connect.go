package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/socks"
	"fmt"
	"net"
	"time"
)

// TCPRelay handles a SOCKS5 CONNECT request on the slave side,
// establishing a TCP connection to the requested destination.
type TCPRelay struct {
	ctx     context.Context
	m       msg.SocksConnect
	sessCtl ClientControlSession
	deps    *config.Dependencies
	logger  *log.Logger
}

// NewTCPRelay creates a new TCP relay for handling a SOCKS5 CONNECT request.
func NewTCPRelay(ctx context.Context, m msg.SocksConnect, sessCtl ClientControlSession, logger *log.Logger, deps *config.Dependencies) *TCPRelay {
	return &TCPRelay{
		ctx:     ctx,
		m:       m,
		sessCtl: sessCtl,
		deps:    deps,
		logger:  logger,
	}
}

// Handle establishes a TCP connection to the target destination and relays
// data between the SOCKS5 client (via control session) and the destination.
func (tr *TCPRelay) Handle() error {
	connRemote, err := tr.sessCtl.GetOneChannelContext(tr.ctx)
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %s", err)
	}
	defer connRemote.Close()

	addr := format.Addr(tr.m.RemoteHost, tr.m.RemotePort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		if isErrorHostUnreachable(err) {
			if err := socks.WriteReplyError(connRemote, socks.ReplyHostUnreachable); err != nil {
				return fmt.Errorf("writing Reply error response (network unreachable): %s", err)
			}

			return nil
		}

		if err := socks.WriteReplyError(connRemote, socks.ReplyGeneralFailure); err != nil {
			return fmt.Errorf("writing Reply error response (general failure): %s", err)
		}

		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	// Get the TCP dialer function from dependencies or use default
	dialerFn := config.GetTCPDialerFunc(tr.deps)
	conn, err := dialerFn(tr.ctx, "tcp", nil, tcpAddr)
	if err != nil {
		if isErrorConnectionRefused(err) {
			if err := socks.WriteReplyError(connRemote, socks.ReplyConnectionRefused); err != nil {
				return fmt.Errorf("writing Reply error response (connection refused): %s", err)
			}

			return nil
		}

		if isErrorNetworkUnreachable(err) {
			if err := socks.WriteReplyError(connRemote, socks.ReplyNetworkUnreachable); err != nil {
				return fmt.Errorf("writing Reply error response (network unreachable): %s", err)
			}

			return nil
		}

		if err := socks.WriteReplyError(connRemote, socks.ReplyGeneralFailure); err != nil {
			return fmt.Errorf("writing Reply error response (general failure): %s", err)
		}

		return fmt.Errorf("net.Dial(tcp, %s): %s", addr, err)
	}
	defer conn.Close()

	if c, ok := connRemote.(interface{ SetWriteDeadline(time.Time) error }); ok {
		_ = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
		defer c.SetWriteDeadline(time.Time{})
	}
	if err := socks.WriteReplySuccessConnect(connRemote, conn.LocalAddr()); err != nil {
		return fmt.Errorf("socks.WriteReplySuccess(): %s", err)
	}

	// Try to enable keep-alive if it's a TCP connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
	}

	pipeio.Pipe(tr.ctx, connRemote, conn, func(err error) {
		tr.logger.ErrorMsg("Handling connect to %s: %s\n", addr, err)
	})

	return nil
}
