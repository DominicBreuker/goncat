package msg

import "encoding/gob"

func init() {
	gob.Register(PortFwd{})
}

// PortFwd ...
type PortFwd struct {
	LocalHost string
	LocalPort int

	RemoteHost string
	RemotePort int
}

// MsgType ...
func (m PortFwd) MsgType() string {
	return "PortFwd"
}
