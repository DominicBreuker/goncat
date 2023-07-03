package msg

type Message interface {
	MsgType() string
}
