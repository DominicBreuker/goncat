package ws

import (
	"context"
	"crypto/tls"
	"dominicbreuker/goncat/pkg/config"
	"fmt"
	"net"
	"net/http"

	"github.com/coder/websocket"
)

type Dialer struct {
	ctx stringCtx
	url string
}

type stringCtx = context.Context

func NewDialer(ctx context.Context, addr string, proto config.Protocol) *Dialer {
	return &Dialer{
		ctx: ctx,
		url: fmt.Sprintf("%s://%s", proto.String(), addr),
	}
}

func (d *Dialer) Dial(ctx context.Context) (net.Conn, error) {
	opts := &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	}
	// For wss, skip verification; inner TLS (app layer) is authoritative.
	// Leaving it enabled for ws is harmless.
	opts.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	c, _, err := websocket.Dial(ctx, d.url, opts)
	if err != nil {
		return nil, fmt.Errorf("websocket.Dial(%s): %w", d.url, err)
	}
	return websocket.NetConn(ctx, c, websocket.MessageBinary), nil
}
