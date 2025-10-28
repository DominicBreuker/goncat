package entrypoint

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"net"
)

// masterHandler is the master handler function.
type masterHandler func(context.Context, *config.Shared, *config.Master, net.Conn) error

// slaveHandler is the slave handler function signature (runs the handler directly
// and returns its final error). This mirrors masterHandler.
type slaveHandler func(context.Context, *config.Shared, net.Conn) error
