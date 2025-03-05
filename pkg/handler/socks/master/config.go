package master

import "fmt"

// Config ...
type Config struct {
	LocalHost string
	LocalPort int
}

func (cfg Config) String() string {
	return fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort)
}
