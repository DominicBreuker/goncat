// Package msg defines message types for communication between master and slave
// sessions. Messages are serialized using gob encoding and passed over control
// channels to coordinate operations like command execution, port forwarding,
// and SOCKS proxy functionality.
package msg

// Message is the interface that all message types must implement.
// MsgType returns a string identifier for the message type, used for
// debugging and logging purposes.
type Message interface {
	MsgType() string
}
