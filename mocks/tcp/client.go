package tcp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

// DialFunc is a minimal dial function used by the test client.
// It matches the signature of MockTCPNetwork.DialTCP.
type DialFunc func(network string, laddr, raddr *net.TCPAddr) (net.Conn, error)

// Client is a simple line-oriented test client that connects using a provided
// dial function (so tests can pass the mock network's dialer). It dials on
// creation and provides WriteLine, ReadLine and Close.
type Client struct {
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer
	mu   sync.Mutex // protects writes
}

// NewClient creates and returns a connected Client using the provided dial function.
// network is typically "tcp" and addr is the address string to resolve (e.g. "127.0.0.1:8000").
func NewClient(dial DialFunc, network, addr string) (*Client, error) {
	if dial == nil {
		return nil, fmt.Errorf("dial func is nil")
	}
	if addr == "" {
		return nil, fmt.Errorf("remote address is empty")
	}

	raddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return nil, err
	}

	conn, err := dial(network, nil, raddr)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn: conn,
		r:    bufio.NewReader(conn),
		w:    bufio.NewWriter(conn),
	}

	return c, nil
}

// WriteLine writes a single line (appends a newline if none) and flushes.
func (c *Client) WriteLine(line string) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("client not connected")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if !strings.HasSuffix(line, "\n") {
		line = line + "\n"
	}

	if _, err := c.w.WriteString(line); err != nil {
		return err
	}
	return c.w.Flush()
}

// ReadLine reads a single line from the server (without the trailing newline).
func (c *Client) ReadLine() (string, error) {
	if c == nil || c.conn == nil {
		return "", fmt.Errorf("client not connected")
	}
	s, err := c.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
