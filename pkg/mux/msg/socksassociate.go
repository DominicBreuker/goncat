package msg

import "encoding/gob"

func init() {
	gob.Register(SocksAssociate{})
}

// SocksAssociate represents a message instructing the slave to create a UDP
// relay for SOCKS5 UDP ASSOCIATE operations. This enables UDP traffic forwarding
// through the SOCKS proxy.
type SocksAssociate struct{}

// MsgType returns the message type identifier for SocksAssociate messages.
func (m SocksAssociate) MsgType() string {
	return "SocksAssociate"
}
