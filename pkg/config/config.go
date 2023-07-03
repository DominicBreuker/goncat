package config

import "fmt"

type Shared struct {
	Host    string
	Port    int
	SSL     bool
	Key     string
	Verbose bool
}

var KeySalt = "98263df478dbb76e25eed7e71750e59dbffcb1f401413472f9b128f10bb3cc01af3942a17980a24cd1a26bd3ab87a0fec835faf59aa4f1a1dc7f2416c5765e9e" // overwrite with custom value during release build

func (c *Shared) Validate() []error {
	var errors []error

	if !c.SSL && c.Key != "" {
		errors = append(errors, fmt.Errorf("You must use '--ssl' to use '--key'"))
	}

	if c.Port < 1 || c.Port > 65535 {
		errors = append(errors, fmt.Errorf("'--port' must be in [1, 65535]"))
	}

	return errors
}

func (c *Shared) GetKey() string {
	if c.Key == "" {
		return ""
	}

	return KeySalt + c.Key
}
