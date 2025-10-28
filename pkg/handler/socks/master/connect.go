// pkg/handler/socks/master/connect.go
package master

import (
	"bufio"
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/socks"
	"fmt"
	"io"
	"net"
	"time"
)

func (srv *Server) handleConnect(connLocal net.Conn, sr *socks.Request) error {
	// Ask slave to connect
	m := msg.SocksConnect{
		RemoteHost: sr.DstAddr.String(),
		RemotePort: int(sr.DstPort),
	}

	opCtx, cancel := context.WithTimeout(srv.ctx, 10*time.Second)
	defer cancel()

	connRemote, err := srv.sessCtl.SendAndGetOneChannelContext(opCtx, m)
	if err != nil {
		// best-effort error reply to client
		bw := bufio.NewReadWriter(bufio.NewReader(connLocal), bufio.NewWriter(connLocal))
		_ = socks.WriteReplyError(bw, socks.ReplyGeneralFailure)
		_ = bw.Flush()
		return fmt.Errorf("SendAndGetOneChannel() for conn: %s", err)
	}
	// connRemote carries the SOCKS reply emitted by the slave
	// We'll read it explicitly and forward it before switching to raw piping.
	// We also close on exit.
	defer connRemote.Close()

	// 1) Read CONNECT reply header (VER, REP, RSV, ATYP) from slave
	rRemote := bufio.NewReader(connRemote)
	wLocal := bufio.NewWriter(connLocal)
	header := make([]byte, 4)
	if _, err := io.ReadFull(rRemote, header); err != nil {
		// Couldn’t get a reply; tell client general failure if possible
		_ = socks.WriteReplyError(bufio.NewReadWriter(bufio.NewReader(connLocal), wLocal), socks.ReplyGeneralFailure)
		_ = wLocal.Flush()
		return fmt.Errorf("reading CONNECT response header from slave: %w", err)
	}

	// Forward header to client
	if _, err := wLocal.Write(header); err != nil {
		return fmt.Errorf("forwarding CONNECT header to client: %w", err)
	}

	// Determine how many more bytes to read (BND.ADDR + BND.PORT)
	atyp := header[3]
	var addrLen int
	switch socks.Atyp(atyp) {
	case socks.AddressTypeIPv4:
		addrLen = 4
	case socks.AddressTypeIPv6:
		addrLen = 16
	case socks.AddressTypeFQDN:
		// length-prefixed FQDN
		lb, err := rRemote.ReadByte()
		if err != nil {
			return fmt.Errorf("reading FQDN len from slave: %w", err)
		}
		if err := wLocal.WriteByte(lb); err != nil {
			return fmt.Errorf("forwarding FQDN len to client: %w", err)
		}
		addrLen = int(lb)
	default:
		return fmt.Errorf("unexpected ATYP: %d", atyp)
	}

	// Read remaining address + 2-byte port and forward
	remain := make([]byte, addrLen+2)
	if _, err := io.ReadFull(rRemote, remain); err != nil {
		return fmt.Errorf("reading CONNECT remain from slave: %w", err)
	}
	if _, err := wLocal.Write(remain); err != nil {
		return fmt.Errorf("forwarding CONNECT remain to client: %w", err)
	}
	if err := wLocal.Flush(); err != nil {
		return fmt.Errorf("flushing CONNECT reply to client: %w", err)
	}

	// If reply code is not success, do NOT start piping.
	if header[1] != byte(socks.ReplySuccess) {
		// Slave reported an error; we're done.
		return nil
	}

	// 2) Now that the client has the success reply, flip to raw piping
	pipeio.Pipe(srv.ctx, connLocal, connRemote, func(err error) {
		srv.cfg.Logger.ErrorMsg("Pipe(stdio, conn): %s\n", err)
	})

	return nil
}
