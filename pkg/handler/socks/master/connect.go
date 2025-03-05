package master

import (
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/socks"
	"fmt"
	"net"
)

func (srv *Server) handleConnect(connLocal net.Conn, sr *socks.Request) error {
	m := msg.SocksConnect{
		RemoteHost: sr.DstAddr.String(),
		RemotePort: int(sr.DstPort),
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
