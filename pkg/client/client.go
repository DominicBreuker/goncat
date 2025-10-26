// Package client provides functionality for establishing network connections
// with support for multiple protocols (TCP, WebSocket) and optional TLS encryption.
package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"dominicbreuker/goncat/pkg/transport/tcp"
	"dominicbreuker/goncat/pkg/transport/udp"
	"dominicbreuker/goncat/pkg/transport/ws"
)

// dependencies holds the injectable dependencies for testing.
type dependencies struct {
	newTCPDialer func(string, *config.Dependencies) (transport.Dialer, error)
	newWSDialer  func(context.Context, string, config.Protocol) transport.Dialer
	newUDPDialer func(string, time.Duration) (transport.Dialer, error)
	tlsUpgrader  func(net.Conn, string, time.Duration) (net.Conn, error)
}

// Client manages a network connection with support for multiple transport protocols
// and optional TLS encryption with mutual authentication.
type Client struct {
	ctx context.Context
	cfg *config.Shared

	conn net.Conn
}

// New creates a new Client with the given context and configuration.
func New(ctx context.Context, cfg *config.Shared) *Client {
	return &Client{
		ctx: ctx,
		cfg: cfg,
	}
}

// Close closes the client's network connection and logs the closure.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// GetConnection returns the underlying network connection.
func (c *Client) GetConnection() net.Conn {
	return c.conn
}

// Connect establishes a connection to the configured remote address.
// It supports TCP, WebSocket, and UDP protocols, and optionally upgrades to TLS.
// The connection is stored in the Client and can be retrieved via GetConnection.
func (c *Client) Connect() error {
	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			return tcp.NewDialer(addr, deps)
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			return ws.NewDialer(ctx, addr, proto)
		},
		newUDPDialer: func(addr string, timeout time.Duration) (transport.Dialer, error) {
			return udp.NewDialer(addr, timeout)
		},
		tlsUpgrader: upgradeToTLS,
	}
	return c.connect(deps)
}

// connect is the internal implementation that accepts injected dependencies for testing.
func (c *Client) connect(deps *dependencies) error {
	addr := format.Addr(c.cfg.Host, c.cfg.Port)

	log.InfoMsg("Connecting to %s\n", addr)

	var d transport.Dialer
	var err error
	switch c.cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		d = deps.newWSDialer(c.ctx, addr, c.cfg.Protocol)
	case config.ProtoUDP:
		// UDP/QUIC handles transport-level TLS internally (like WebSocket wss)
		// Application-level TLS (--ssl) will be applied after connection if needed
		d, err = deps.newUDPDialer(addr, c.cfg.Timeout)
		if err != nil {
			return fmt.Errorf("create udp dialer: %w", err)
		}
	default:
		d, err = deps.newTCPDialer(addr, c.cfg.Deps)
		if err != nil {
			return fmt.Errorf("create dialer: %w", err)
		}
	}

	c.conn, err = d.Dial(c.ctx)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	// Apply application-level TLS upgrade if --ssl is set
	// This happens for all transports: TCP, WS, WSS, and UDP
	// (WS and UDP already have transport-level TLS, but app-level TLS is separate)
	if c.cfg.SSL {
		c.conn, err = deps.tlsUpgrader(c.conn, c.cfg.GetKey(), c.cfg.Timeout)
		if err != nil {
			return fmt.Errorf("upgrade to tls: %w", err)
		}
	}

	return nil
}

// upgradeToTLS wraps the given connection with TLS encryption.
// If a key is provided, it enables mutual authentication using generated certificates.
// The function configures TLS 1.3 as the minimum version.
func upgradeToTLS(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // custom verification below
	}

	if key != "" {
		caCert, cert, err := crypto.GenerateCertificates(key)
		if err != nil {
			return nil, fmt.Errorf("generate certificates: %w", err)
		}

		cfg.Certificates = []tls.Certificate{cert} // client Cert
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return customVerifier(caCert, rawCerts)
		}
	}

	tlsConn := tls.Client(conn, cfg)

	// set a handshake deadline to avoid blocking indefinitely
	if timeout > 0 {
		_ = tlsConn.SetDeadline(time.Now().Add(timeout))
		defer func() { _ = tlsConn.SetDeadline(time.Time{}) }()
	}
	if err := tlsConn.Handshake(); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("tls handshake: %w", err)
	}

	return tlsConn, nil
}

// customVerifier verifies the certificate but cares only about the root certificate, not SANs
func customVerifier(caCert *x509.CertPool, rawCerts [][]byte) error {
	if len(rawCerts) != 1 {
		return fmt.Errorf("unexpected number of raw certs: %d", len(rawCerts))
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	if _, err := cert.Verify(x509.VerifyOptions{
		Roots: caCert,
	}); err != nil {
		return fmt.Errorf("verify certificate: %w", err)
	}

	return nil
}
