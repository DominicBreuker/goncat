package msg

import "encoding/gob"

func init() {
	gob.Register(Foreground{})
}

// Foreground represents a message instructing the slave to execute a program
// or create a data pipe in the foreground. When Exec is empty, data is simply
// piped without executing a program. When Pty is true, the slave creates a
// pseudo-terminal for interactive sessions.
type Foreground struct {
	Exec string // Program to execute, leave empty to just pipe data
	Pty  bool   // Ask slave to create a PTY
}

// MsgType returns the message type identifier for Foreground messages.
func (m Foreground) MsgType() string {
	return "Foreground"
}
