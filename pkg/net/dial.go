// Package net provides simplified network connection APIs for goncat.
// It replaces the previous pkg/client and pkg/server packages with
// two simple functions: Dial and ListenAndServe.
package net

import (
	"context"
	"fmt"
	"net"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/format"
)

// Dial establishes a connection to the configured remote address.
// It supports TCP, WebSocket, and UDP protocols with optional TLS encryption.
// The function handles all connection setup, TLS upgrades, timeout management,
// and cleanup internally. Returns the established connection or an error.
//
// The context can be used to cancel the dial operation at any time.
// Timeouts for individual operations are controlled by cfg.Timeout.
func Dial(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
	deps := &dialDependencies{
		dialTCP: realDialTCP,
		dialWS:  realDialWS,
		dialWSS: realDialWSS,
		dialUDP: realDialUDP,
	}
	return dial(ctx, cfg, deps)
}

// dial is the internal implementation that accepts injected dependencies for testing.
func dial(ctx context.Context, cfg *config.Shared, deps *dialDependencies) (net.Conn, error) {
	addr := format.Addr(cfg.Host, cfg.Port)

	cfg.Logger.InfoMsg("Connecting to %s\n", addr)
	cfg.Logger.VerboseMsg("Dialing %s using protocol %s", addr, cfg.Protocol)

	// Step 1: Establish the connection with proper timeout handling
	conn, err := establishConnection(ctx, cfg, deps)
	if err != nil {
		return nil, fmt.Errorf("establishing connection: %w", err)
	}

	// Step 2: Upgrade to TLS if requested
	if cfg.SSL {
		cfg.Logger.VerboseMsg("Upgrading connection to TLS")
		tlsConn, err := upgradeTLS(conn, cfg)
		if err != nil {
			_ = conn.Close() // Close the original connection, not the nil tlsConn
			return nil, fmt.Errorf("upgrading to TLS: %w", err)
		}
		cfg.Logger.VerboseMsg("TLS upgrade completed")
		conn = tlsConn // Only assign if successful
	}

	return conn, nil
}
