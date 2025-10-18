// Package ws provides WebSocket transport implementations.
// It implements the transport.Dialer and transport.Listener interfaces
// for WebSocket (ws://) and secure WebSocket (wss://) connections.
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

// Dialer implements the transport.Dialer interface for WebSocket connections.
type Dialer struct {
	ctx context.Context
	url string
}

// NewDialer creates a new WebSocket dialer for the specified address and protocol.
// The proto parameter determines whether to use ws:// or wss://.
func NewDialer(ctx context.Context, addr string, proto config.Protocol) *Dialer {
	return &Dialer{
		ctx: ctx,
		url: fmt.Sprintf("%s://%s", proto.String(), addr),
	}

}

// Dial establishes a WebSocket connection to the configured URL.
// It returns a net.Conn that wraps the WebSocket connection for binary message exchange.
// Accepts a context to allow cancellation.
func (d *Dialer) Dial(ctx context.Context) (net.Conn, error) {
	c, _, err := websocket.Dial(ctx, d.url, &websocket.DialOptions{
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

	return websocket.NetConn(ctx, c, websocket.MessageBinary), nil
}
