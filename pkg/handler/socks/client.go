package socks

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"dominicbreuker/goncat/pkg/socks"
	"fmt"
	"net"
	"strings"
)

// Client ...
type Client struct {
	ctx     context.Context
	m       msg.SocksConnect
	sessCtl ClientControlSession
}

// ClientControlSession ...
type ClientControlSession interface {
	GetOneChannel() (net.Conn, error)
}

// NewClient ...
func NewClient(ctx context.Context, m msg.SocksConnect, sessCtl ClientControlSession) *Client {
	return &Client{
		ctx:     ctx,
		m:       m,
		sessCtl: sessCtl,
	}
}

// Handle ...
func (h *Client) Handle() error {
	connRemote, err := h.sessCtl.GetOneChannel()
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %s", err)
	}
	defer connRemote.Close()

	if isIPv6(h.m.RemoteHost) {
		// net.ResolveTCPAddr likes to see IPv6 addresses with brackets
		h.m.RemoteHost = ensureIPv6Brackets(h.m.RemoteHost)
	}
	addr := fmt.Sprintf("%s:%d", h.m.RemoteHost, h.m.RemotePort)

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

	if err := socks.WriteReplySuccess(connRemote, connLocal.LocalAddr()); err != nil {
		return fmt.Errorf("socks.WriteReplySuccess(): %s", err)
	}

	connLocal.SetKeepAlive(true)

	pipeio.Pipe(h.ctx, connRemote, connLocal, func(err error) {
		log.ErrorMsg("Handling connect to %s: %s", addr, err)
	})

	return nil
}

// isErrorHostUnreachable is my personal list of Go errors that shall count as the SOCKS5 "Host unreachable" error
func isErrorHostUnreachable(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "no such host")
}

// isErrorConnectionRefused is my personal list of Go errors that shall count as the SOCKS5 "Connection Refused" error
func isErrorConnectionRefused(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "connection refused") ||
		strings.HasSuffix(s, "host is down")
}

// isErrorNetworkUnreachable is my personal list of Go errors that shall count as the SOCKS5 "Network unreachable" error
func isErrorNetworkUnreachable(err error) bool {
	s := err.Error()
	return strings.HasSuffix(s, "network is unreachable")
}

func isIPv6(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To16() != nil
}

func ensureIPv6Brackets(s string) string {
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s
	}
	return "[" + s + "]"
}
