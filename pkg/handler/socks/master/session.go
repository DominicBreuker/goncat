package master

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"net"
)

// ServerControlSession represents the interface for communicating over
// a multiplexed control session to handle SOCKS5 proxy requests.
type ServerControlSession interface {
	SendAndGetOneChannel(m msg.Message) (net.Conn, error)
	Send(m msg.Message) error
}
