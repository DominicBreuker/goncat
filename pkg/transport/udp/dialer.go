// Package udp provides UDP transport implementations with KCP reliability.
// It implements the transport.Dialer and transport.Listener interfaces
// for UDP network connections using the KCP protocol for reliable delivery.
package udp

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"fmt"
	"net"

	kcp "github.com/xtaci/kcp-go/v5"
)

// Dialer implements the transport.Dialer interface for UDP connections with KCP.
type Dialer struct {
	remoteAddr   *net.UDPAddr
	packetConnFn config.PacketListenerFunc
}

// NewDialer creates a new UDP dialer for the specified address.
// The deps parameter is optional and can be nil to use default implementations.
func NewDialer(addr string, deps *config.Dependencies) (*Dialer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveUDPAddr(udp, %s): %w", addr, err)
	}

	packetConnFn := config.GetPacketListenerFunc(deps)

	return &Dialer{
		remoteAddr:   udpAddr,
		packetConnFn: packetConnFn,
	}, nil
}

// Dial establishes a KCP session over UDP to the configured address.
// It accepts a context so the caller can cancel the dial.
func (d *Dialer) Dial(ctx context.Context) (net.Conn, error) {
	// Create UDP packet connection using stdlib
	// Use ":0" for local address to let OS choose an ephemeral port
	conn, err := d.packetConnFn("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("net.ListenPacket(udp, :0): %w", err)
	}

	// Upgrade to KCP session using kcp.NewConn
	// Parameters: remoteAddr, block cipher (nil for no encryption), dataShards (0), parityShards (0), conn
	kcpConn, err := kcp.NewConn(d.remoteAddr.String(), nil, 0, 0, conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("kcp.NewConn(%s): %w", d.remoteAddr.String(), err)
	}

	// Configure KCP for optimal performance
	// SetNoDelay(nodelay, interval, resend, nc)
	// nodelay: 0=disable, 1=enable
	// interval: internal update interval in ms
	// resend: 0=disable fast resend, 1=enable fast resend, 2=2 ACK crosses trigger fast resend
	// nc: 0=normal congestion control, 1=disable congestion control
	kcpConn.SetNoDelay(1, 10, 2, 1)
	kcpConn.SetStreamMode(true)
	kcpConn.SetWindowSize(1024, 1024)

	return kcpConn, nil
}
