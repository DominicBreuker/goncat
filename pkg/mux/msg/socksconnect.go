package msg

import "encoding/gob"

func init() {
	gob.Register(SocksConnect{})
}

// SocksConnect ...
type SocksConnect struct {
	RemoteHost string
	RemotePort int
}

// MsgType ...
func (m SocksConnect) MsgType() string {
	return "SocksConnect"
}
