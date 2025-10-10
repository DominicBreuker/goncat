// Package master provides the master-side implementation of a SOCKS5 proxy server.
// It listens for SOCKS5 client connections and forwards requests through the
// multiplexed control session to the slave.
package master

import "fmt"

// Config contains the configuration for the SOCKS5 proxy server on the master side.
type Config struct {
	LocalHost string // Local host address to bind the SOCKS5 server to
	LocalPort int    // Local port to bind the SOCKS5 server to
}

func (cfg Config) String() string {
	return fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort)
}
