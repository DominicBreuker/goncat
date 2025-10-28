package portfwd

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"fmt"
	"net"
)

// Client handles establishing connections to remote destinations
// in response to port forwarding requests from the control session.
type Client struct {
	ctx         context.Context
	m           msg.Connect
	sessCtl     ClientControlSession
	tcpDialerFn config.TCPDialerFunc
	udpDialerFn config.UDPDialerFunc
	logger      *log.Logger
}

// ClientControlSession represents the interface for obtaining a channel
// from the control session for forwarding data.
type ClientControlSession interface {
	GetOneChannelContext(ctx context.Context) (net.Conn, error)
}

// NewClient creates a new port forwarding client that will connect to
// the destination specified in the message.
// The deps parameter is optional and can be nil to use default implementations.
func NewClient(ctx context.Context, m msg.Connect, sessCtl ClientControlSession, logger *log.Logger, deps *config.Dependencies) *Client {
	return &Client{
		ctx:         ctx,
		m:           m,
		sessCtl:     sessCtl,
		tcpDialerFn: config.GetTCPDialerFunc(deps),
		udpDialerFn: config.GetUDPDialerFunc(deps),
		logger:      logger,
	}
}

// Handle establishes a connection to the remote destination and pipes
// data between it and the channel obtained from the control session.
func (h *Client) Handle() error {
	protocol := h.m.Protocol
	if protocol == "" {
		protocol = "tcp" // default to TCP for backward compatibility
	}

	switch protocol {
	case "tcp":
		return h.handleTCP()
	case "udp":
		return h.handleUDP()
	default:
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// handleTCP handles TCP port forwarding by establishing a TCP connection
// and piping data bidirectionally.
func (h *Client) handleTCP() error {
	connRemote, err := h.sessCtl.GetOneChannelContext(h.ctx)
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %w", err)
	}
	defer connRemote.Close()

	addr := format.Addr(h.m.RemoteHost, h.m.RemotePort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %w", addr, err)
	}

	connLocal, err := h.tcpDialerFn(h.ctx, "tcp", nil, tcpAddr)
	if err != nil {
		return fmt.Errorf("net.Dial(tcp, %s): %w", addr, err)
	}
	defer connLocal.Close()

	// Try to enable keep-alive if it's a TCP connection
	if tcpConn, ok := connLocal.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
	}

	pipeio.Pipe(h.ctx, connRemote, connLocal, func(err error) {
		h.logger.ErrorMsg("Handling connect to %s: %w", addr, err)
	})

	return nil
}

// handleUDP handles UDP port forwarding by establishing a UDP connection
// and forwarding datagrams bidirectionally through the yamux stream.
func (h *Client) handleUDP() error {

	connRemote, err := h.sessCtl.GetOneChannelContext(h.ctx)
	if err != nil {
		return fmt.Errorf("GetOneChannelContext(): %w", err)
	}
	defer connRemote.Close()

	addr := format.Addr(h.m.RemoteHost, h.m.RemotePort)

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveUDPAddr(udp, %s): %w", addr, err)
	}

	connLocal, err := h.udpDialerFn(h.ctx, "udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("dial(udp, %s): %w", addr, err)
	}
	defer connLocal.Close()

	// Create channels for coordinating goroutines
	done := make(chan struct{})
	errCh := make(chan error, 2)

	// Forward data from yamux stream to UDP socket
	go func() {
		buffer := make([]byte, 65536)
		for {
			select {
			case <-h.ctx.Done():
				return
			case <-done:
				return
			default:
			}

			n, err := connRemote.Read(buffer)
			if err != nil {
				if h.ctx.Err() != nil {
					return
				}
				errCh <- fmt.Errorf("read from stream: %w", err)
				return
			}

			// Use WriteTo for compatibility with both connected and unconnected UDP sockets
			_, err = connLocal.WriteTo(buffer[:n], udpAddr)
			if err != nil {
				errCh <- fmt.Errorf("write to UDP: %w", err)
				return
			}
		}
	}()

	// Forward data from UDP socket to yamux stream
	go func() {
		buffer := make([]byte, 65536)
		for {
			select {
			case <-h.ctx.Done():
				return
			case <-done:
				return
			default:
			}

			n, _, err := connLocal.ReadFrom(buffer)
			if err != nil {
				if h.ctx.Err() != nil {
					return
				}
				errCh <- fmt.Errorf("read from UDP: %w", err)
				return
			}

			_, err = connRemote.Write(buffer[:n])
			if err != nil {
				errCh <- fmt.Errorf("write to stream: %w", err)
				return
			}
		}
	}()

	// Wait for error or context cancellation
	select {
	case err := <-errCh:
		close(done)
		return err
	case <-h.ctx.Done():
		close(done)
		return nil
	}
}
