package msg

import "encoding/gob"

func init() {
	gob.Register(Connect{})
}

// Connect ...
type Connect struct {
	RemoteHost string
	RemotePort int
}

// MsgType ...
func (m Connect) MsgType() string {
	return "Connect"
}
