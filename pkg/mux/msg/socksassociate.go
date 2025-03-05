package msg

import "encoding/gob"

func init() {
	gob.Register(SocksAssociate{})
}

// SocksRelayCreate ...
type SocksAssociate struct{}

// MsgType ...
func (m SocksAssociate) MsgType() string {
	return "SocksAssociate"
}
