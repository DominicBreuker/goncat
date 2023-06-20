package client

import (
	"crypto/tls"
	"crypto/x509"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/crypto"
	"dominicbreuker/goncat/pkg/exec"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux"
	"dominicbreuker/goncat/pkg/terminal"
	"fmt"
	"net"
)

// Client ...
type Client struct {
	cfg config.Config
}

// New ...
func New(cfg config.Config) *Client {
	return &Client{
		cfg: cfg,
	}
}

// Connect ...
func (c *Client) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)

	var conn net.Conn
	var err error
	if c.cfg.SSL {
		conn, err = dialTLS(addr, c.cfg.GetKey())
	} else {
		conn, err = dialTCP(addr)
	}
	if err != nil {
		return fmt.Errorf("dial: %s", err)
	}

	defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())

	if c.cfg.Pty {
		c.handleWithPTY(conn)
	} else {
		c.handlePlain(conn)
	}

	return nil
}

func (c *Client) handleWithPTY(conn net.Conn) error {
	connCtl, connData, err := mux.OpenChannels(conn)
	if err != nil {
		return fmt.Errorf("mux.OpenChannels(conn): %s", err)
	}
	defer connCtl.Close()
	defer connData.Close()
	defer conn.Close()

	if c.cfg.LogFile != "" {
		var err error
		connData, err = log.NewLoggedConn(connData, c.cfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", c.cfg.LogFile, err)
		}
	}

	if c.cfg.Exec != "" {
		if err := exec.RunWithPTY(connCtl, connData, c.cfg.Exec, c.cfg.Verbose); err != nil {
			return fmt.Errorf("exec.RunWithPTY(...): %s", err)
		}
	} else {
		if err := terminal.PipeWithPTY(connCtl, connData, c.cfg.Verbose); err != nil {
			return fmt.Errorf("terminal.PipeWithPTY(connCtl, connData): %s", err)
		}
	}

	return nil
}

func (c *Client) handlePlain(conn net.Conn) error {
	if c.cfg.LogFile != "" {
		var err error
		conn, err = log.NewLoggedConn(conn, c.cfg.LogFile)
		if err != nil {
			return fmt.Errorf("enabling logging to %s: %s", c.cfg.LogFile, err)
		}
	}

	if c.cfg.Exec != "" {
		if err := exec.Run(conn, c.cfg.Exec); err != nil {
			return fmt.Errorf("exec.Run(conn, %s): %s", c.cfg.Exec, err)
		}
	} else {
		terminal.Pipe(conn, c.cfg.Verbose)
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
