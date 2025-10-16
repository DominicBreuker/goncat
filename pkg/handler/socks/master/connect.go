package master

import (
	"context"
	"time"

	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/socks"
	"fmt"
	"net"
)

// handleConnect processes a SOCKS5 CONNECT request by establishing a connection
// through the control session and piping data between the client and destination.
func (srv *Server) handleConnect(connLocal net.Conn, sr *socks.Request) error {
	m := msg.SocksConnect{
		RemoteHost: sr.DstAddr.String(),
		RemotePort: int(sr.DstPort),
	}

	// Bound the control operation with a short timeout so a stalled control
	// session doesn't block the handler indefinitely.
	opCtx, cancel := context.WithTimeout(srv.ctx, 10*time.Second)
	defer cancel()

	connRemote, err := srv.sessCtl.SendAndGetOneChannelContext(opCtx, m)
	if err != nil {
		return fmt.Errorf("SendAndGetOneChannel() for conn: %s", err)
	}
	defer connRemote.Close()

	pipeio.Pipe(srv.ctx, connLocal, connRemote, func(err error) {
		log.ErrorMsg("Pipe(stdio, conn): %s\n", err)
	})

	return nil
}
