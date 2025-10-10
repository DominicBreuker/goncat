package msg

import (
	"encoding/gob"
	"fmt"
)

func init() {
	gob.Register(SocksDatagram{})
}

// SocksDatagram represents a UDP datagram to be forwarded through a SOCKS5
// UDP relay. It contains the destination address, port, and payload data.
// This is used for SOCKS5 UDP ASSOCIATE operations.
type SocksDatagram struct {
	Addr string
	Port int
	Data []byte
}

// MsgType returns the message type identifier for SocksDatagram messages.
func (m SocksDatagram) MsgType() string {
	return "SocksDatagram"
}

// String returns a human-readable representation of the datagram for debugging.
func (m SocksDatagram) String() string {
	return fmt.Sprintf("Datagram[%d|%s|%d|%s]", len(m.Data), m.Addr, m.Port, m.Data)
}
