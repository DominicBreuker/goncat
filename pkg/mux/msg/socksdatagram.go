package msg

import (
	"encoding/gob"
	"fmt"
)

func init() {
	gob.Register(SocksDatagram{})
}

// SocksDatagram ...
type SocksDatagram struct {
	Addr string
	Port int
	Data []byte
}

// MsgType ...
func (m SocksDatagram) MsgType() string {
	return "SocksDatagram"
}

func (m SocksDatagram) String() string {
	return fmt.Sprintf("Datagram[%d|%s|%d|%s]", len(m.Data), m.Addr, m.Port, m.Data)
}
