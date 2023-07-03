package config

import "fmt"

type Shared struct {
	Host    string
	Port    int
	SSL     bool
	Key     string
	Verbose bool
}

var KeySalt = "bn6ySqbg2BgmHaljx3mhg94DOybkBF3G" // overwrite with custom value during release build

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
