package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/socks"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"time"
)

// UDPRelay manages UDP datagram forwarding for SOCKS5 ASSOCIATE requests.
// It relays UDP packets between the local SOCKS5 client and the remote destination.
type UDPRelay struct {
	ctx  context.Context
	Conn net.PacketConn // Changed from *net.UDPConn to support mocking

	ConnRemote  net.Conn
	readRemote  *gob.Decoder
	writeRemote *gob.Encoder

	cancel context.CancelFunc

	ClientIP   netip.Addr
	ClientPort uint16

	// Server reference for logger access
	srv *Server
}

// NewUDPRelay creates a new UDP relay for handling SOCKS5 UDP ASSOCIATE requests.
// It binds a local UDP port and sets up communication with the remote end.
func NewUDPRelay(ctx context.Context, addr string, sr *socks.Request, connRemote net.Conn, srv *Server, deps *config.Dependencies) (*UDPRelay, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr+":")
	if err != nil {
		return nil, fmt.Errorf("net.ResolveUDPAddr(udp, %s:): %s", addr, err)
	}

	// Get the UDP listener function from dependencies or use default
	listenerFn := config.GetUDPListenerFunc(deps)
	conn, err := listenerFn("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("ListenUDP(udp, %s): %s", udpAddr, err)
	}

	// we serve a relay but must ensure its lifetime is bound to the TCP connection
	// solution: close UDP relay once TCP connection closes, and ignore all errors from then on
	rCtx, cancel := context.WithCancel(ctx)

	return &UDPRelay{
		ctx:  rCtx,
		Conn: conn,

		ConnRemote:  connRemote,
		readRemote:  gob.NewDecoder(connRemote),
		writeRemote: gob.NewEncoder(connRemote),

		cancel: cancel,

		// if unknown the SocksRequest contains zero values
		ClientIP:   sr.DstAddr.ToNetipAddr(),
		ClientPort: sr.DstPort,
	}, nil
}

// Close shuts down the UDP relay and closes the underlying connection.
func (r *UDPRelay) Close() error {
	r.cancel()
	return r.Conn.Close()
}

// LogError logs an error message with a prefix to indicate where it comes from
func (r *UDPRelay) LogError(format string, a ...interface{}) {
	r.srv.cfg.Logger.ErrorMsg("UDP Relay: "+format, a...)
}

// RemoteToLocal reads UDP datagrams received from the remote and and sends them to the local client
func (r *UDPRelay) RemoteToLocal() {
	data := make(chan *msg.SocksDatagram)

	go func() {
		defer close(data)
		defer r.Close()

		for {
			p, err := r.readFromRemote()
			if err != nil {
				if err == io.EOF {
					return
				}
				if r.ctx.Err() != nil {
					return // cancelled
				}

				r.srv.cfg.Logger.ErrorMsg("Receiving next datagram: %s\n", err)
				return
			}

			data <- p
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			return
		case p, ok := <-data:
			if !ok {
				return
			}
			if err := r.sendToLocal(p); err != nil {
				r.srv.cfg.Logger.ErrorMsg("SOCKS error: sending packet to local end: %s\n", err)
				return
			}
		}
	}
}

// readFromRemote reads a UDP datagram message from the remote end.
func (r *UDPRelay) readFromRemote() (*msg.SocksDatagram, error) {
	p := msg.SocksDatagram{}

	// Allow ctx cancellation to interrupt blocking Decode by setting a read
	// deadline on the remote connection when r.ctx is done. Use a done
	// channel to avoid leaking a goroutine.
	done := make(chan struct{})
	go func() {
		select {
		case <-r.ctx.Done():
			if c, ok := r.ConnRemote.(interface{ SetReadDeadline(time.Time) error }); ok {
				_ = c.SetReadDeadline(time.Now())
			}
		case <-done:
		}
	}()

	err := r.readRemote.Decode(&p)
	close(done)

	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}

		return nil, fmt.Errorf("reading datagram: %s", err)
	}

	return &p, nil
}

// sendToLocal sends a UDP datagram to the local SOCKS5 client.
func (r *UDPRelay) sendToLocal(data *msg.SocksDatagram) error {
	if (r.ClientIP != netip.Addr{}) && r.ClientPort != 0 {
		// Use the existing WriteUDPRequest helper for PacketConn
		err := writeUDPRequest(r.Conn, r.ClientIP, r.ClientPort, data.Data)
		if err != nil {
			return fmt.Errorf("writing to local conn: %s", err)
		}
	} else {
		r.srv.cfg.Logger.InfoMsg("Received datagram but not sent data back to local client since address not known\n", data.Data)
	}

	return nil
}

// writeUDPRequest writes a SOCKS5 UDP datagram to a PacketConn.
// This is similar to socks.WriteUDPRequestAddrPort but works with PacketConn.
func writeUDPRequest(conn net.PacketConn, ip netip.Addr, port uint16, data []byte) error {
	// Build the SOCKS5 UDP request header
	// Format: RSV RSV FRAG ATYP ADDR PORT DATA
	var packet []byte
	packet = append(packet, socks.RSV, socks.RSV, socks.FRAG) // RSV, RSV, FRAG=0

	// Unmap IPv4-mapped IPv6 addresses to pure IPv4
	ip = ip.Unmap()

	if ip.Is4() {
		packet = append(packet, byte(socks.AddressTypeIPv4))
		ipBytes := ip.As4()
		packet = append(packet, ipBytes[:]...)
	} else if ip.Is6() {
		packet = append(packet, byte(socks.AddressTypeIPv6))
		ipBytes := ip.As16()
		packet = append(packet, ipBytes[:]...)
	} else {
		return fmt.Errorf("IP %s was neither IPv4 nor IPv6", ip)
	}

	// Add port (network byte order, big-endian)
	packet = append(packet, byte(port>>8), byte(port))

	// Add data payload
	packet = append(packet, data...)

	// Write to the client using PacketConn
	clientAddr := &net.UDPAddr{
		IP:   net.IP(ip.AsSlice()),
		Port: int(port),
	}
	_, err := conn.WriteTo(packet, clientAddr)
	return err
}

// LocalToRemote reads UDP datagrams from the local SOCKS5 client and forwards them
// to the remote destination through the control session.
func (r *UDPRelay) LocalToRemote() {
	data := make(chan []byte)

	go func() {
		defer close(data)
		defer r.Close()

		buff := make([]byte, 65507)
		for {
			// Set a short read deadline so we can check r.ctx for cancellation periodically.
			if udpConn, ok := r.Conn.(interface{ SetReadDeadline(time.Time) error }); ok {
				udpConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			}

			n, clientAddr, err := r.Conn.ReadFrom(buff)
			// Read the manual before handling errors: https://pkg.go.dev/net#PacketConn
			// ... Callers should always process the n > 0 bytes returned before considering the error err...

			// Extract IP and port from the client address
			if udpAddr, ok := clientAddr.(*net.UDPAddr); ok {
				// Update client IP if it's not set or is unspecified (0.0.0.0 or ::)
				if !r.ClientIP.IsValid() || r.ClientIP.IsUnspecified() {
					addr, ok := netip.AddrFromSlice(udpAddr.IP)
					if ok {
						r.ClientIP = addr
					}
				}
				if r.ClientPort == 0 {
					r.ClientPort = uint16(udpAddr.Port)
				}
			}

			if n > 0 {
				b := make([]byte, n)
				copy(b, buff[:n])
				data <- b
			}

			// now we check if there was an error during read
			if err != nil {
				// if deadline exceeded, continue the loop so we can re-check context
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					if r.ctx.Err() != nil {
						return
					}
					continue
				}

				if r.ctx.Err() == context.Canceled {
					return // errors expected since we closed connection
				}

				r.srv.cfg.Logger.ErrorMsg("SOCKS UDP Relay: reading from local conn: %s\n", err)
				return
			}
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			return
		case b, ok := <-data:
			if !ok {
				return
			}

			if err := r.sendToRemote(b); err != nil {
				r.srv.cfg.Logger.ErrorMsg("SOCKS UDP Relay: sending to remote end: %s\n", err)
				return
			}
		}
	}
}

// sendToRemote parses a SOCKS5 UDP datagram and sends it to the remote end.
func (r *UDPRelay) sendToRemote(buf []byte) error {
	d, err := socks.ReadUDPDatagram(buf)
	if err != nil {
		// no support for optional FRAG, see https://datatracker.ietf.org/doc/html/rfc1928#section-7
		if errors.Is(err, socks.ErrFragmentationNotSupported) {
			r.srv.cfg.Logger.InfoMsg("SOCKS UDP Relay: datagram dropped since fragmentation was requested but is not supported")
			return nil
		}

		return fmt.Errorf("parsing datagram: %s", err)
	}

	m := msg.SocksDatagram{
		Addr: d.DstAddr.String(),
		Port: int(d.DstPort),
		Data: d.Data,
	}

	if err := r.writeRemote.Encode(&m); err != nil {
		return fmt.Errorf("encoding datagram: %s", err)
	}

	return nil
}
