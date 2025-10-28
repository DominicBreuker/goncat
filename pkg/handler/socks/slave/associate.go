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
	"sync"
	"time"
)

// UDPRelay manages UDP datagram forwarding for SOCKS5 ASSOCIATE requests on the slave side.
// It receives datagrams from the master and forwards them to destination hosts via UDP.
type UDPRelay struct {
	ctx context.Context

	ConnLocal  net.PacketConn
	ConnRemote net.Conn

	cancel   context.CancelFunc
	mu       sync.RWMutex
	isClosed bool
	sessCtl  ClientControlSession
	logger   *log.Logger
}

// NewUDPRelay creates a new UDP relay for handling SOCKS5 ASSOCIATE requests.
// It binds a local UDP port for sending/receiving datagrams and opens a control channel.
func NewUDPRelay(ctx context.Context, sessCtl ClientControlSession, logger *log.Logger, deps *config.Dependencies) (*UDPRelay, error) {
	// Get the packet listener function from dependencies or use default
	listenerFn := config.GetPacketListenerFunc(deps)
	connLocal, err := listenerFn("udp", "0.0.0.0:")
	if err != nil {
		return nil, fmt.Errorf("ListenPacket(udp, 0.0.0.0:): %s", err)
	}

	connRemote, err := sessCtl.GetOneChannelContext(ctx)
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
		logger:     logger,
	}, nil
}

// Close shuts down the UDP relay and closes all connections.
func (r *UDPRelay) Close() error {
	r.mu.Lock()
	r.isClosed = true
	r.mu.Unlock()

	r.cancel()
	defer r.ConnRemote.Close()
	return r.ConnLocal.Close()
}

// closed checks if the relay has been closed (thread-safe).
func (r *UDPRelay) closed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isClosed
}

// LogError logs an error message with a prefix to indicate where it comes from.
func (r *UDPRelay) LogError(format string, a ...interface{}) {
	r.logger.ErrorMsg("UDP Relay: "+format, a...)
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

	type udpPacket struct {
		data []byte
		addr *net.UDPAddr
	}
	data := make(chan udpPacket)

	go func() {
		defer close(data)
		defer r.Close()

		buff := make([]byte, 65507)
		for {
			if r.closed() {
				return
			}

			// Short read deadline to periodically unblock and check context cancellation.
			if udpConn, ok := r.ConnLocal.(interface{ SetReadDeadline(time.Time) error }); ok {
				udpConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			}

			n, remoteAddr, err := r.ConnLocal.ReadFrom(buff)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					if r.ctx.Err() != nil {
						return
					}
					continue
				}

				if r.closed() {
					return // ignore errors if closed
				}

				r.LogError("receiving packet from %s: %s\n", remoteAddr, err)
				return
			}

			// Extract UDP address
			udpAddr, ok := remoteAddr.(*net.UDPAddr)
			if !ok {
				continue
			}

			// Copy data and send to channel
			b := make([]byte, n)
			copy(b, buff[:n])
			select {
			case data <- udpPacket{data: b, addr: udpAddr}:
			case <-r.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			return
		case pkt, ok := <-data:
			if !ok {
				return
			}

			if err := writeRemote.Encode(&msg.SocksDatagram{
				Addr: pkt.addr.IP.String(),
				Port: pkt.addr.Port,
				Data: pkt.data,
			}); err != nil {
				if r.closed() {
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
	if r.closed() {
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
