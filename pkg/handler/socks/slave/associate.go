package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

// UDPRelay manages UDP datagram forwarding for SOCKS5 ASSOCIATE requests on the slave side.
// It receives datagrams from the master and forwards them to destination hosts via UDP.
type UDPRelay struct {
	ctx context.Context

	ConnLocal  net.PacketConn
	ConnRemote net.Conn

	cancel   context.CancelFunc
	isClosed bool
	sessCtl  ClientControlSession
}

// NewUDPRelay creates a new UDP relay for handling SOCKS5 ASSOCIATE requests.
// It binds a local UDP port for sending/receiving datagrams and opens a control channel.
func NewUDPRelay(ctx context.Context, sessCtl ClientControlSession, deps *config.Dependencies) (*UDPRelay, error) {
	// Get the packet listener function from dependencies or use default
	listenerFn := config.GetPacketListenerFunc(deps)
	connLocal, err := listenerFn("udp", "0.0.0.0:")
	if err != nil {
		return nil, fmt.Errorf("ListenPacket(udp, 0.0.0.0:): %s", err)
	}

	connRemote, err := sessCtl.GetOneChannel()
	if err != nil {
		defer connLocal.Close()
		return nil, fmt.Errorf("AcceptNewChannel(): %s", err)
	}

	rCtx, cancel := context.WithCancel(ctx)

	return &UDPRelay{
		ctx:        rCtx,
		ConnLocal:  connLocal,
		ConnRemote: connRemote,
		cancel:     cancel,
		isClosed:   false,
		sessCtl:    sessCtl,
	}, nil
}

// Close shuts down the UDP relay and closes all connections.
func (r *UDPRelay) Close() error {
	r.isClosed = true

	r.cancel()
	defer r.ConnRemote.Close()
	return r.ConnLocal.Close()
}

// LogError logs an error message with a prefix to indicate where it comes from
func (r *UDPRelay) LogError(format string, a ...interface{}) {
	log.ErrorMsg("UDP Relay: "+format, a...)
}

// Serve starts the UDP relay, forwarding datagrams between local and remote ends.
// It blocks until the remote connection closes or an error occurs.
func (r *UDPRelay) Serve() error {
	go r.localToRemote()     // forward to remote forever
	return r.remoteToLocal() // read from remote until it closes connection
}

// localToRemote reads UDP datagrams from local destinations and forwards them to the remote end.
func (r *UDPRelay) localToRemote() {
	writeRemote := gob.NewEncoder(r.ConnRemote)

	data := make(chan []byte)

	go func() {
		defer close(data)
		defer r.Close()

		buff := make([]byte, 65507)
		for {
			if r.isClosed {
				return
			}

			n, remoteAddr, err := r.ConnLocal.ReadFrom(buff)
			if err != nil {
				if r.isClosed {
					return // ignore errors if closed
				}

				r.LogError("receiving packet from %s: %s", remoteAddr, err)
				return
			}

			select {
			case data <- buff[:n]:
			case <-r.ctx.Done():
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

			if err := writeRemote.Encode(&msg.SocksDatagram{
				Data: b,
			}); err != nil {
				if r.isClosed {
					return // ignore errors if closed
				}

				r.LogError("encoding packet: %s", err)
			}
		}
	}
}

// remoteToLocal reads datagrams from the remote end and forwards them to destination hosts.
func (r *UDPRelay) remoteToLocal() error {
	read := gob.NewDecoder(r.ConnRemote)

	data := make(chan msg.SocksDatagram)
	go func() {
		defer close(data)
		defer r.Close()

		var p msg.SocksDatagram
		for {
			err := read.Decode(&p)
			if err != nil {
				if err == io.EOF {
					return
				}
				if r.ctx.Err() != nil {
					return // cancelled or error
				}

				r.LogError("reading from remote: %s", err)
				return
			}

			select {
			case <-r.ctx.Done():
				return
			case data <- p:
			}
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			if r.ctx.Err() == context.Canceled {
				return nil
			}
			return r.ctx.Err()
		case p, ok := <-data:
			if !ok {
				return nil
			}

			if err := r.sendToDst(p.Addr, p.Port, p.Data); err != nil {
				return fmt.Errorf("delivering UDP traffic: %s", err)
			}
		}
	}
}

// sendToDst sends a UDP datagram to the specified destination address and port.
func (r *UDPRelay) sendToDst(addr string, port int, data []byte) error {
	if r.isClosed {
		return fmt.Errorf("use of closed relay")
	}

	dstAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return fmt.Errorf("failed to resolve address %s:%d", addr, port)
	}

	_, err = r.ConnLocal.WriteTo(data, dstAddr)
	if err != nil {
		return fmt.Errorf("failed to send UDP data: %s", err)
	}

	return nil
}
