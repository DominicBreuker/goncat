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

// Client handles establishing connections to remote destinations
// in response to port forwarding requests from the control session.
type Client struct {
	ctx     context.Context
	m       msg.Connect
	sessCtl ClientControlSession
}

// ClientControlSession represents the interface for obtaining a channel
// from the control session for forwarding data.
type ClientControlSession interface {
	GetOneChannel() (net.Conn, error)
}

// NewClient creates a new port forwarding client that will connect to
// the destination specified in the message.
func NewClient(ctx context.Context, m msg.Connect, sessCtl ClientControlSession) *Client {
	return &Client{
		ctx:     ctx,
		m:       m,
		sessCtl: sessCtl,
	}
}

// Handle establishes a connection to the remote destination and pipes
// data between it and the channel obtained from the control session.
func (h *Client) Handle() error {
	connRemote, err := h.sessCtl.GetOneChannel()
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %s", err)
	}
	defer connRemote.Close()

	addr := format.Addr(h.m.RemoteHost, h.m.RemotePort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	connLocal, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return fmt.Errorf("net.Dial(tcp, %s): %s", addr, err)
	}
	defer connLocal.Close()

	connLocal.SetKeepAlive(true)

	pipeio.Pipe(h.ctx, connRemote, connLocal, func(err error) {
		log.ErrorMsg("Handling connect to %s: %s", addr, err)
	})

	return nil
}
