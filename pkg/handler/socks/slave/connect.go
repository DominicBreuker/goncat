package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/socks"
	"fmt"
	"net"
)

// TCPRelay ...
type TCPRelay struct {
	ctx     context.Context
	m       msg.SocksConnect
	sessCtl ClientControlSession
}

// NewTCPRelay ...
func NewTCPRelay(ctx context.Context, m msg.SocksConnect, sessCtl ClientControlSession) *TCPRelay {
	return &TCPRelay{
		ctx:     ctx,
		m:       m,
		sessCtl: sessCtl,
	}
}

// Handle ...
func (tr *TCPRelay) Handle() error {
	connRemote, err := tr.sessCtl.GetOneChannel()
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

	connLocal, err := net.DialTCP("tcp", nil, tcpAddr)
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
	defer connLocal.Close()

	if err := socks.WriteReplySuccessConnect(connRemote, connLocal.LocalAddr()); err != nil {
		return fmt.Errorf("socks.WriteReplySuccess(): %s", err)
	}

	connLocal.SetKeepAlive(true)

	pipeio.Pipe(tr.ctx, connRemote, connLocal, func(err error) {
		log.ErrorMsg("Handling connect to %s: %s", addr, err)
	})

	return nil
}
