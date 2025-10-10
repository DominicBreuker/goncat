package msg

import "encoding/gob"

func init() {
	gob.Register(SocksConnect{})
}

// SocksConnect represents a message instructing the slave to establish a TCP
// connection to a remote host and port as part of a SOCKS5 CONNECT operation.
// This is used for proxying TCP traffic through the SOCKS proxy.
type SocksConnect struct {
	RemoteHost string
	RemotePort int
}

// MsgType returns the message type identifier for SocksConnect messages.
func (m SocksConnect) MsgType() string {
	return "SocksConnect"
}
