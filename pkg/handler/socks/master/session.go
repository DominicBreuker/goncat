package master

import (
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"net"
)

// ServerControlSession represents the interface for communicating over
// a multiplexed control session to handle SOCKS5 proxy requests.
type ServerControlSession interface {
	SendAndGetOneChannelContext(ctx context.Context, m msg.Message) (net.Conn, error)
	// SendContext sends a control message that respects ctx cancellation.
	SendContext(ctx context.Context, m msg.Message) error
}
