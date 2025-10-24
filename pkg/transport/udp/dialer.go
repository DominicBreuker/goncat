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
	// Determine network type based on remote address family
	network := "udp4"
	if d.remoteAddr.IP.To4() == nil {
		// IPv6 remote address
		network = "udp6"
	}

	conn, err := d.packetConnFn(network, ":0")
	if err != nil {
		return nil, fmt.Errorf("net.ListenPacket(%s, :0): %w", network, err)
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
	kcpConn.SetWindowSize(1024, 1024)

	// IMPORTANT: KCP requires an explicit write to establish the session.
	// The kcp.NewConn() function creates a client-side session object, but doesn't
	// actually send any packets until data is written. The server's AcceptKCP()
	// will block until it receives the first packet from the client.
	// We write a single dummy byte here to trigger the KCP handshake, allowing
	// the server to accept the connection. The server's listener will read and
	// discard this byte before passing the connection to the handler.
	_, err = kcpConn.Write([]byte{0})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("initial write to trigger KCP handshake: %w", err)
	}

	return kcpConn, nil
}
