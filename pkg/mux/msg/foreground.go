package msg

import "encoding/gob"

func init() {
	gob.Register(Foreground{})
}

// Foreground ...
type Foreground struct {
	Exec string // Program to execute, leave empty to just pipe data
	Pty  bool   // Ask slave to create a PTY
}

// MsgType ...
func (m Foreground) MsgType() string {
	return "Foreground"
}
