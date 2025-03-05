package master

import (
	"dominicbreuker/goncat/pkg/mux/msg"
	"net"
)

// ServerControlSession ...
type ServerControlSession interface {
	SendAndGetOneChannel(m msg.Message) (net.Conn, error)
	Send(m msg.Message) error
}
