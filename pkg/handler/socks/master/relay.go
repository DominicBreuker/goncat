package master

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/socks"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
)

// UDPRelay ...
type UDPRelay struct {
	ctx  context.Context
	Conn *net.UDPConn

	ConnRemote  net.Conn
	readRemote  *gob.Decoder
	writeRemote *gob.Encoder

	cancel context.CancelFunc

	ClientIP   netip.Addr
	ClientPort uint16
}

// NewUDPRelay ...
func NewUDPRelay(ctx context.Context, addr string, sr *socks.Request, connRemote net.Conn) (*UDPRelay, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr+":")
	if err != nil {
		return nil, fmt.Errorf("net.ResolveUDPAddr(udp, %s:): %s", addr, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.ListenUDP(udp, %s): %s", udpAddr, err)
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

// Close ...
func (r *UDPRelay) Close() error {
	r.cancel()
	return r.Conn.Close()
}

// LogError logs an error message with a prefix to indicate where it comes from
func (r *UDPRelay) LogError(format string, a ...interface{}) {
	log.ErrorMsg("UDP Relay: "+format, a...)
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

				log.ErrorMsg("Receiving next datagram: %s\n", err)
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
				log.ErrorMsg("SOCKS error: sending packet to local end: %s\n", err)
				return
			}
		}
	}
}

func (r *UDPRelay) readFromRemote() (*msg.SocksDatagram, error) {
	p := msg.SocksDatagram{}

	err := r.readRemote.Decode(&p)
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}

		return nil, fmt.Errorf("reading datagram: %s", err)
	}

	return &p, nil
}

func (r *UDPRelay) sendToLocal(data *msg.SocksDatagram) error {
	if (r.ClientIP != netip.Addr{}) && r.ClientPort != 0 {
		err := socks.WriteUDPRequestAddrPort(r.Conn, r.ClientIP, r.ClientPort, data.Data)
		if err != nil {
			return fmt.Errorf("writing to local conn: %s", err)
		}
	} else {
		log.InfoMsg("Received datagram but not sent data back to local client since address not known\n", data.Data)
	}

	return nil
}

// LocalToRemote ...
func (r *UDPRelay) LocalToRemote() {
	data := make(chan []byte)

	go func() {
		defer close(data)
		defer r.Close()

		buff := make([]byte, 65507)
		for {
			n, clientAddr, err := r.Conn.ReadFromUDPAddrPort(buff)
			// Read the manual before handling errors: https://pkg.go.dev/net#PacketConn
			// ... Callers should always process the n > 0 bytes returned before considering the error err...
			if (r.ClientIP == netip.Addr{}) {
				r.ClientIP = clientAddr.Addr()
			}
			if r.ClientPort == 0 {
				r.ClientPort = clientAddr.Port()
			}

			if n > 0 {
				b := make([]byte, n)
				copy(b, buff[:n])
				data <- b
			}

			// now we check if there was an error during read
			if err != nil {
				if r.ctx.Err() == context.Canceled {
					return // errros expected since we closed connection
				}

				log.ErrorMsg("SOCKS UDP Relay: reading from local conn: %s\n", err)
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
				log.ErrorMsg("SOCKS UDP Relay: sending to remote end: %s\n", err)
				return
			}
		}
	}
}

func (r *UDPRelay) sendToRemote(buf []byte) error {
	d, err := socks.ReadUDPDatagram(buf)
	if err != nil {
		// no support for optional FRAG, see https://datatracker.ietf.org/doc/html/rfc1928#section-7
		if errors.Is(err, socks.ErrFragmentationNotSupported) {
			log.InfoMsg("SOCKS UDP Relay: datagram dropped since fragmentation was requested but is not supported")
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
