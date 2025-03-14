package config

import "fmt"

// Shared ...
type Shared struct {
	Protocol Protocol
	Host     string
	Port     int
	SSL      bool
	Key      string
	Verbose  bool
}

type Protocol int

const (
	ProtoTCP = 1
	ProtoWS  = 2
	ProtoWSS = 3
)

func (p Protocol) String() string {
	switch p {
	case ProtoTCP:
		return "tcp"
	case ProtoWS:
		return "ws"
	case ProtoWSS:
		return "wss"
	default:
		return ""
	}
}

// KeySalt ...
var KeySalt = "98263df478dbb76e25eed7e71750e59dbffcb1f401413472f9b128f10bb3cc01af3942a17980a24cd1a26bd3ab87a0fec835faf59aa4f1a1dc7f2416c5765e9e" // overwrite with custom value during release build

// Validate ...
func (c *Shared) Validate() []error {
	var errors []error

	if !c.SSL && c.Key != "" {
		errors = append(errors, fmt.Errorf("you must use '--ssl' to use '--key'"))
	}

	if err := validatePort(c.Port); err != nil {
		errors = append(errors, fmt.Errorf("'--port': %s", err))
	}

	return errors
}

// GetKey ...
func (c *Shared) GetKey() string {
	if c.Key == "" {
		return ""
	}

	return KeySalt + c.Key
}
