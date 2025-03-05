package client

import (
	"crypto/tls"
	"crypto/x509"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/format"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"net"
)

// Client ...
type Client struct {
	cfg *config.Shared

	conn net.Conn
}

// New ...
func New(cfg *config.Shared) *Client {
	return &Client{
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

	var err error
	if c.cfg.SSL {
		c.conn, err = dialTLS(addr, c.cfg.GetKey())
	} else {
		c.conn, err = dialTCP(addr)
	}
	if err != nil {
		return fmt.Errorf("dial: %s", err)
	}

	return nil
}

func dialTCP(addr string) (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("net.Dial(tcp, %s): %s", addr, err)
	}

	conn.SetKeepAlive(true)
	return conn, nil
}

func dialTLS(addr string, key string) (net.Conn, error) {
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

	cfg := &tls.Config{}
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

	conn, err := tls.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("tls.Dial(tcp): %s", err)
	}

	return conn, nil
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
