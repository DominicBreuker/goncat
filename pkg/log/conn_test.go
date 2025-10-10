package log

import (
	"bytes"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// mockConn implements net.Conn for testing
type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func newMockConn() *mockConn {
	return &mockConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (m *mockConn) Read(b []byte) (int, error) {
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (int, error) {
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9090}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestNewLoggedConn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file system test in short mode")
	}

	tmpFile := t.TempDir() + "/test.log"
	conn := newMockConn()

	loggedConn, err := NewLoggedConn(conn, tmpFile)
	if err != nil {
		t.Fatalf("NewLoggedConn() error = %v", err)
	}
	if loggedConn == nil {
		t.Fatal("NewLoggedConn() returned nil")
	}

	// Verify log file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("NewLoggedConn() did not create log file")
	}
}

func TestLoggedConn_Write(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file system test in short mode")
	}

	tmpFile := t.TempDir() + "/test.log"
	conn := newMockConn()

	loggedConn, err := NewLoggedConn(conn, tmpFile)
	if err != nil {
		t.Fatalf("NewLoggedConn() error = %v", err)
	}

	testData := []byte("test data")
	n, err := loggedConn.Write(testData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(testData))
	}

	// Verify data was written to log file
	logData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(logData, testData) {
		t.Errorf("Log file contains %q, want %q", logData, testData)
	}
}

func TestLoggedConn_Read(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file system test in short mode")
	}

	tmpFile := t.TempDir() + "/test.log"
	conn := newMockConn()
	testData := []byte("read test data")
	conn.readBuf.Write(testData)

	loggedConn, err := NewLoggedConn(conn, tmpFile)
	if err != nil {
		t.Fatalf("NewLoggedConn() error = %v", err)
	}

	buf := make([]byte, len(testData))
	n, err := loggedConn.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Read() read %d bytes, want %d", n, len(testData))
	}

	// Verify data was logged
	logData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(logData, testData) {
		t.Errorf("Log file contains %q, want %q", logData, testData)
	}
}

func TestLoggedConn_Addresses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file system test in short mode")
	}

	tmpFile := t.TempDir() + "/test.log"
	conn := newMockConn()

	loggedConn, err := NewLoggedConn(conn, tmpFile)
	if err != nil {
		t.Fatalf("NewLoggedConn() error = %v", err)
	}

	if loggedConn.LocalAddr() == nil {
		t.Error("LocalAddr() returned nil")
	}
	if loggedConn.RemoteAddr() == nil {
		t.Error("RemoteAddr() returned nil")
	}
}

func TestLoggedConn_Deadlines(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file system test in short mode")
	}

	tmpFile := t.TempDir() + "/test.log"
	conn := newMockConn()

	loggedConn, err := NewLoggedConn(conn, tmpFile)
	if err != nil {
		t.Fatalf("NewLoggedConn() error = %v", err)
	}

	deadline := time.Now().Add(time.Second)

	if err := loggedConn.SetDeadline(deadline); err != nil {
		t.Errorf("SetDeadline() error = %v", err)
	}
	if err := loggedConn.SetReadDeadline(deadline); err != nil {
		t.Errorf("SetReadDeadline() error = %v", err)
	}
	if err := loggedConn.SetWriteDeadline(deadline); err != nil {
		t.Errorf("SetWriteDeadline() error = %v", err)
	}
}

func TestLoggedConn_Read_EOF(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file system test in short mode")
	}

	tmpFile := t.TempDir() + "/test.log"
	conn := newMockConn()
	// Empty buffer will return EOF

	loggedConn, err := NewLoggedConn(conn, tmpFile)
	if err != nil {
		t.Fatalf("NewLoggedConn() error = %v", err)
	}

	buf := make([]byte, 10)
	_, err = loggedConn.Read(buf)
	if err != io.EOF {
		t.Errorf("Read() error = %v, want EOF", err)
	}
}
