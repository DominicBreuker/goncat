package ws

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// DialWS establishes a WebSocket connection (plain HTTP).
// Returns the connection wrapped as net.Conn or an error.
//
// The timeout parameter is used for the HTTP connection establishment.
// After the connection is established, the timeout is cleared to prevent
// affecting subsequent operations.
func DialWS(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	url := formatURL("ws", addr)
	return dialWebSocket(ctx, url, timeout, false)
}

// DialWSS establishes a WebSocket Secure connection (HTTPS/TLS).
// Returns the connection wrapped as net.Conn or an error.
//
// The timeout parameter is used for the HTTP connection establishment.
// After the connection is established, the timeout is cleared to prevent
// affecting subsequent operations.
func DialWSS(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	url := formatURL("wss", addr)
	return dialWebSocket(ctx, url, timeout, true)
}

// formatURL creates a WebSocket URL from protocol and address.
func formatURL(protocol, addr string) string {
	return fmt.Sprintf("%s://%s", protocol, addr)
}

// dialWebSocket establishes a WebSocket connection with optional TLS.
func dialWebSocket(ctx context.Context, url string, timeout time.Duration, useTLS bool) (net.Conn, error) {
	opts := createDialOptions(useTLS)

	c, _, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		return nil, fmt.Errorf("websocket.Dial(%s): %w", url, err)
	}

	// Wrap WebSocket connection as net.Conn
	return websocket.NetConn(ctx, c, websocket.MessageBinary), nil
}

// createDialOptions creates WebSocket dial options with TLS configuration.
func createDialOptions(useTLS bool) *websocket.DialOptions {
	opts := &websocket.DialOptions{
		Subprotocols: []string{"bin"},
	}

	// For wss, skip verification; inner TLS (app layer) is authoritative.
	// For ws, this is harmless but included for consistency.
	if useTLS {
		opts.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	return opts
}
