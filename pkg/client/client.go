package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/transport"
	"dominicbreuker/goncat/pkg/transport/tcp"
	"dominicbreuker/goncat/pkg/transport/ws"
	"fmt"
	"net"
)

// Client ...
type Client struct {
	ctx context.Context
	cfg *config.Shared

	conn net.Conn
}

// New ...
func New(ctx context.Context, cfg *config.Shared) *Client {
	return &Client{
		ctx: ctx,
		cfg: cfg,
	}
}

// Close ...
func (c *Client) Close() error {
	log.InfoMsg("Connection to %s closed\n", c.conn.RemoteAddr())

	return c.conn.Close()
}

// GetConnection ...
func (c *Client) GetConnection() net.Conn {
	return c.conn
}

// Connect ...
func (c *Client) Connect() error {
	addr := format.Addr(c.cfg.Host, c.cfg.Port)

	log.InfoMsg("Connecting to %s\n", addr)

	var d transport.Dialer
	var err error
	switch c.cfg.Protocol {
	case config.ProtoWS, config.ProtoWSS:
		d, err = ws.NewDialer(c.ctx, addr, c.cfg.Protocol), nil
	default:
		d, err = tcp.NewDialer(addr)
	}
	if err != nil {
		return fmt.Errorf("NewDialer: %s", err)
	}

	c.conn, err = d.Dial()
	if err != nil {
		return fmt.Errorf("Dial(): %s", err)
	}

	if c.cfg.SSL {
		c.conn, err = upgradeToTLS(c.conn, c.cfg.GetKey())
		if err != nil {
			return fmt.Errorf("upgradeToTLS: %s", err)
		}
	}

	return nil
}

func upgradeToTLS(conn net.Conn, key string) (net.Conn, error) {
	setTCPKeepAlive := func(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {
		if tcpConn, ok := clientHello.Conn.(*net.TCPConn); ok {
			if err := tcpConn.SetKeepAlive(true); err != nil {
				return nil, fmt.Errorf("conn.SetKeepAlive(true): %s", err)
			}
		} else {
			return nil, fmt.Errorf("conn.SetKeepAlive(true): no TCP connection")
		}

		return nil, nil
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}
	cfg.GetConfigForClient = setTCPKeepAlive
	cfg.InsecureSkipVerify = true // we implement ourselves to skip hostname validation

	if key != "" {
		caCert, cert, err := crypto.GenerateCertificates(key)
		if err != nil {
			return nil, fmt.Errorf("crypto.GenerateCertificates(%s): %s", key, err)
		}

		cfg.Certificates = []tls.Certificate{cert} // client Cert
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return customVerifier(caCert, rawCerts)
		}
	}

	tlsConn := tls.Client(conn, cfg)
	tlsConn.Handshake()

	return tlsConn, nil
}

// customVerifier verifies the certificate but cares only about the root certificate, not SANs
func customVerifier(caCert *x509.CertPool, rawCerts [][]byte) error {
	if len(rawCerts) != 1 {
		return fmt.Errorf("unexpected number of rawCerts: %d", len(rawCerts))
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("x509.ParseCertificate(rawCert): %s", err)
	}

	if _, err := cert.Verify(x509.VerifyOptions{
		Roots: caCert,
	}); err != nil {
		return fmt.Errorf("cert.Verify(caCert): %s", err)
	}

	return nil
}
