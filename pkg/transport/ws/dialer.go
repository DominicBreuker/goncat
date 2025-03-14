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

// Dialer ...
type Dialer struct {
	ctx context.Context
	url string
}

// NewDialer ...
func NewDialer(ctx context.Context, addr string, proto config.Protocol) *Dialer {
	return &Dialer{
		ctx: ctx,
		url: fmt.Sprintf("%s://%s", proto.String(), addr),
	}

}

// Dial ...
func (d *Dialer) Dial() (net.Conn, error) {
	c, _, err := websocket.Dial(d.ctx, d.url, &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		Subprotocols: []string{"bin"},
	})
	if err != nil {
		return nil, fmt.Errorf("websocket.Dial(%s): %s", d.url, err)
	}

	return websocket.NetConn(d.ctx, c, websocket.MessageBinary), nil
}
