package msg

import "encoding/gob"

func init() {
	gob.Register(Hello{})
}

// Hello is a message sent by the master/slave to identify itself upon session establishment to the other side.
type Hello struct {
	ID string // Identifier of the connecting slave
}

// MsgType returns the message type identifier for Hello messages.
func (m Hello) MsgType() string {
	return "Hello"
}
