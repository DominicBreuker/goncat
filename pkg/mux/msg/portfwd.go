package msg

import "encoding/gob"

func init() {
	gob.Register(PortFwd{})
}

// PortFwd represents a message configuring port forwarding between local and
// remote endpoints. The slave receives this message and establishes a connection
// to the remote host/port while forwarding traffic to/from the local host/port.
type PortFwd struct {
	Protocol string // "tcp" or "udp", defaults to "tcp" if empty

	LocalHost string
	LocalPort int

	RemoteHost string
	RemotePort int
}

// MsgType returns the message type identifier for PortFwd messages.
func (m PortFwd) MsgType() string {
	return "PortFwd"
}
