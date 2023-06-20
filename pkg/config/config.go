package config

import "fmt"

type Config struct {
	Host    string
	Port    int
	SSL     bool
	Key     string
	Exec    string
	Pty     bool
	LogFile string
	Verbose bool
}

var KeySalt = "bn6ySqbg2BgmHaljx3mhg94DOybkBF3G" // overwrite with custom value during release build

func (c *Config) Validate() []error {
	var errors []error

	if !c.SSL && c.Key != "" {
		errors = append(errors, fmt.Errorf("You must use '--ssl' to use '--key'"))
	}

	if c.Port < 1 || c.Port > 65535 {
		errors = append(errors, fmt.Errorf("'--port' must be in [1, 65535]"))
	}

	return errors
}

func (c *Config) GetKey() string {
	if c.Key == "" {
		return ""
	}

	return KeySalt + c.Key
}
