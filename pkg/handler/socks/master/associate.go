package master

import (
	"bufio"
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/socks"
	"fmt"
	"io"
	"time"
)

// handleAssociate processes a SOCKS5 ASSOCIATE request by setting up a UDP relay
// between the client and the remote destination through the control session.
func (srv *Server) handleAssociate(bufConnLocal *bufio.ReadWriter, sr *socks.Request) error {
	defer bufConnLocal.Flush()

	// establish connection to remote end
	// Bound the control operation with a short timeout so a stalled control
	// session doesn't block the associate handler indefinitely.
	opCtx, cancel := context.WithTimeout(srv.ctx, 10*time.Second)
	defer cancel()

	connRemote, err := srv.sessCtl.SendAndGetOneChannelContext(opCtx, msg.SocksAssociate{})
	if err != nil {
		if err := socks.WriteReplyError(bufConnLocal, socks.ReplyGeneralFailure); err != nil {
			return fmt.Errorf("writing Reply error response: %s", err)
		}

		return fmt.Errorf("getting channel: %s", err)
	}
	defer connRemote.Close() // kills relay on other end

	// create a local UDP relay which binds a port
	relay, err := NewUDPRelay(srv.ctx, srv.cfg.LocalHost, sr, connRemote, srv, srv.cfg.Deps)
	if err != nil {
		if err := socks.WriteReplyError(bufConnLocal, socks.ReplyGeneralFailure); err != nil {
			return fmt.Errorf("writing Reply error response: %s", err)
		}

		return fmt.Errorf("srv.NewUDPRelay(...): %s", err)
	}
	defer relay.Close()

	// confirm successful setup and communicate relay address to local client
	if err := socks.WriteReplySuccessConnect(bufConnLocal, relay.Conn.LocalAddr()); err != nil {
		return fmt.Errorf("socks.WriteReplySuccess(): %s", err)
	}
	if err := bufConnLocal.Flush(); err != nil {
		return fmt.Errorf("flushing socks.WriteReplySuccess(): %s", err)
	}

	go relay.RemoteToLocal() // deliver UDP datagrams from remote end to local client
	go relay.LocalToRemote() // read UDP datagrams from local client and forward to remote end
	// TODO: consider to cancel everything if we get a localToRemote error, in case our client misbehaves

	// wait for the TCP connection to die, which should close the relay (defer)
	buf := make([]byte, 1)
	for {
		if _, err := bufConnLocal.Read(buf); err != nil {
			if err != io.EOF {
				return fmt.Errorf("error reading connLocal: %s", err)
			}

			return nil
		}
	}
}
