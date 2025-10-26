package msg

import "encoding/gob"

func init() {
	gob.Register(Connect{})
}

// Connect represents a message instructing the slave to establish a connection
// to a remote host and port. This is used for port forwarding operations.
type Connect struct {
	Protocol   string // "tcp" or "udp", defaults to "tcp" if empty
	RemoteHost string
	RemotePort int
}

// MsgType returns the message type identifier for Connect messages.
func (m Connect) MsgType() string {
	return "Connect"
}
