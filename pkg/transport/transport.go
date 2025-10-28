// Package transport provides network transport implementations for goncat.
// Each transport (tcp, ws, udp) implements two simple functions instead of interfaces:
//
// Dial Functions:
//   - Establish outbound connections
//   - Accept: context, address, timeout, and optional dependencies
//   - Return: net.Conn or error
//   - Handle all connection setup, timeout management, and cleanup internally
//
// ListenAndServe Functions:
//   - Create listeners and serve connections
//   - Accept: context, address, timeout, handler, and optional dependencies
//   - Return: error (blocks until context cancelled)
//   - Handle all listener setup, connection limiting, timeout management, and cleanup
//
// Transport-specific notes:
//   - TCP: Single Dial and ListenAndServe function with dependencies parameter
//   - WebSocket: Separate functions for ws (plain) and wss (TLS):
//   - DialWS / DialWSS
//   - ListenAndServeWS / ListenAndServeWSS
//   - UDP: Uses QUIC protocol with built-in TLS 1.3:
//   - Single Dial and ListenAndServe function
//   - Init byte handling for stream activation is internal
//
// Timeout Handling:
//   - Timeouts are set before potentially blocking operations
//   - Timeouts are cleared immediately after operations complete
//   - This prevents healthy connections from being killed by lingering timeouts
//
// Example usage:
//
//	// TCP
//	conn, err := tcp.Dial(ctx, "localhost:8080", 10*time.Second, deps)
//	err := tcp.ListenAndServe(ctx, ":8080", 10*time.Second, handler, deps)
//
//	// WebSocket
//	conn, err := ws.DialWS(ctx, "localhost:8080", 10*time.Second)
//	err := ws.ListenAndServeWSS(ctx, ":8443", 10*time.Second, handler)
//
//	// UDP/QUIC
//	conn, err := udp.Dial(ctx, "localhost:12345", 10*time.Second)
//	err := udp.ListenAndServe(ctx, ":12345", 10*time.Second, handler)
package transport

import "net"

// Handler is a function that processes an incoming connection.
// It should handle the connection and return when done.
// The connection will be closed after the handler returns.
type Handler func(net.Conn) error
