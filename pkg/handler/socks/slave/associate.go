package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

// UDPRelay ...
type UDPRelay struct {
	ctx context.Context

	ConnLocal  net.PacketConn
	ConnRemote net.Conn

	cancel   context.CancelFunc
	isClosed bool
	sessCtl  ClientControlSession
}

// NewUDPRelay ...
func NewUDPRelay(ctx context.Context, sessCtl ClientControlSession) (*UDPRelay, error) {
	connLocal, err := net.ListenPacket("udp", "0.0.0.0:")
	if err != nil {
		return nil, fmt.Errorf("net.ListenPacket(udp, 0.0.0.0:): %s", err)
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

// Close ...
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

// Serve ...
func (r *UDPRelay) Serve() error {
	go r.localToRemote()     // forward to remote forever
	return r.remoteToLocal() // read from remote until it closes connection
}

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

	return nil
}

// sendToDst sends data to addr:port via UDP
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
