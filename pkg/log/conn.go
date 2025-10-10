package log

import (
	"fmt"
	"net"
	"os"
	"time"
)

// loggedConn wraps a net.Conn and logs all read/write operations to a file.
type loggedConn struct {
	conn    net.Conn
	logFile *os.File
}

func (lc *loggedConn) Read(b []byte) (int, error) {
	n, err := lc.conn.Read(b)
	if n > 0 {
		_, err = lc.logFile.Write(b[:n])
		if err != nil {
			return 0, fmt.Errorf("reading: %s", err)
		}
	}
	return n, err
}

func (lc *loggedConn) Write(b []byte) (int, error) {
	n, err := lc.conn.Write(b)
	if n > 0 {
		_, err = lc.logFile.Write(b[:n])
		if err != nil {
			return 0, fmt.Errorf("writing: %s", err)
		}
	}
	return n, err
}

func (lc *loggedConn) Close() error {
	return lc.conn.Close()
}

func (lc *loggedConn) LocalAddr() net.Addr {
	return lc.conn.LocalAddr()
}

func (lc *loggedConn) RemoteAddr() net.Addr {
	return lc.conn.RemoteAddr()
}

func (lc *loggedConn) SetDeadline(t time.Time) error {
	return lc.conn.SetDeadline(t)
}

func (lc *loggedConn) SetReadDeadline(t time.Time) error {
	return lc.conn.SetReadDeadline(t)
}

func (lc *loggedConn) SetWriteDeadline(t time.Time) error {
	return lc.conn.SetWriteDeadline(t)
}

// NewLoggedConn wraps a network connection to log all data read from and written to it.
// The log file is created or appended to at the specified path.
func NewLoggedConn(conn net.Conn, logFilePath string) (net.Conn, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &loggedConn{conn: conn, logFile: logFile}, nil
}
