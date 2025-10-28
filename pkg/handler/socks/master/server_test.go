package master

import (
	"bufio"
	"bytes"
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/socks"
	"net"
	"testing"
)

// ServerControlSession interface for testing
type fakeServerControlSession struct {
	sendAndGetOneChannelFn func(m msg.Message) (net.Conn, error)
	sendFn                 func(m msg.Message) error
}

func (f *fakeServerControlSession) SendAndGetOneChannel(m msg.Message) (net.Conn, error) {
	if f.sendAndGetOneChannelFn != nil {
		return f.sendAndGetOneChannelFn(m)
	}
	return nil, nil
}

func (f *fakeServerControlSession) SendAndGetOneChannelContext(ctx context.Context, m msg.Message) (net.Conn, error) {
	return f.SendAndGetOneChannel(m)
}

func (f *fakeServerControlSession) Send(m msg.Message) error {
	if f.sendFn != nil {
		return f.sendFn(m)
	}
	return nil
}

func (f *fakeServerControlSession) SendContext(ctx context.Context, m msg.Message) error {
	if f.sendFn != nil {
		return f.sendFn(m)
	}
	return nil
}

// TestNewServer verifies server creation and initialization.
func TestNewServer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := Config{
		LocalHost: "127.0.0.1",
		LocalPort: 0, // Use ephemeral port
	}
	sessCtl := &fakeServerControlSession{}

	srv, err := NewServer(ctx, cfg, sessCtl)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	defer srv.listener.Close()

	if srv.ctx != ctx {
		t.Error("Server context not set correctly")
	}
	if srv.cfg != cfg {
		t.Error("Server config not set correctly")
	}
	if srv.listener == nil {
		t.Error("Server listener is nil")
	}
}

// TestNewServer_InvalidAddress verifies error handling for invalid addresses.
func TestNewServer_InvalidAddress(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := Config{
		LocalHost: "invalid::address::format",
		LocalPort: 1080,
	}
	sessCtl := &fakeServerControlSession{}

	_, err := NewServer(ctx, cfg, sessCtl)
	if err == nil {
		t.Error("NewServer() expected error with invalid address, got nil")
	}
}

// TestServer_LogError verifies error logging functionality.
func TestServer_LogError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := Config{
		LocalHost: "127.0.0.1",
		LocalPort: 0,
	}
	sessCtl := &fakeServerControlSession{}

	srv, err := NewServer(ctx, cfg, sessCtl)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	defer srv.listener.Close()

	// Test that LogError doesn't panic
	srv.LogError("test error: %s", "value")
}

func TestHandleMethodSelection_NoAuthRequested(t *testing.T) {
	t.Parallel()

	// Create a method selection request with NoAuthenticationRequired
	var buf bytes.Buffer
	bufRW := bufio.NewReadWriter(bufio.NewReader(&buf), bufio.NewWriter(&buf))

	// Write method selection request: version, nmethods, methods
	buf.Write([]byte{
		0x05, // SOCKS version 5
		0x01, // 1 method
		0x00, // NoAuthenticationRequired
	})

	err := handleMethodSelection(bufRW)
	if err != nil {
		t.Errorf("handleMethodSelection() returned unexpected error: %v", err)
	}

	// Read response
	bufRW.Flush()
	response := make([]byte, 2)
	_, err = buf.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response[0] != 0x05 {
		t.Errorf("Expected SOCKS version 5, got %d", response[0])
	}
	if response[1] != 0x00 {
		t.Errorf("Expected method 0 (NoAuth), got %d", response[1])
	}
}

func TestHandleMethodSelection_MultipleMethodsIncludingNoAuth(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bufRW := bufio.NewReadWriter(bufio.NewReader(&buf), bufio.NewWriter(&buf))

	// Write method selection request with multiple methods
	buf.Write([]byte{
		0x05, // SOCKS version 5
		0x03, // 3 methods
		0x01, // GSSAPI
		0x00, // NoAuthenticationRequired
		0x02, // Username/Password
	})

	err := handleMethodSelection(bufRW)
	if err != nil {
		t.Errorf("handleMethodSelection() returned unexpected error: %v", err)
	}

	bufRW.Flush()
	response := make([]byte, 2)
	_, err = buf.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response[1] != 0x00 {
		t.Errorf("Expected method 0 (NoAuth) to be selected, got %d", response[1])
	}
}

func TestHandleMethodSelection_NoAuthNotSupported(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bufRW := bufio.NewReadWriter(bufio.NewReader(&buf), bufio.NewWriter(&buf))

	// Write method selection request without NoAuthenticationRequired
	buf.Write([]byte{
		0x05, // SOCKS version 5
		0x02, // 2 methods
		0x01, // GSSAPI
		0x02, // Username/Password
	})

	err := handleMethodSelection(bufRW)
	if err == nil {
		t.Error("Expected error when NoAuth not requested, got nil")
	}

	bufRW.Flush()
	response := make([]byte, 2)
	_, err = buf.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response[1] != 0xFF {
		t.Errorf("Expected method 0xFF (NoAcceptableMethods), got %d", response[1])
	}
}

func TestHandleMethodSelection_InvalidVersion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bufRW := bufio.NewReadWriter(bufio.NewReader(&buf), bufio.NewWriter(&buf))

	// Write invalid version
	buf.Write([]byte{
		0x04, // SOCKS version 4 (invalid)
		0x01, // 1 method
		0x00, // NoAuthenticationRequired
	})

	err := handleMethodSelection(bufRW)
	if err == nil {
		t.Error("Expected error with invalid SOCKS version, got nil")
	}
}

func TestHandleRequest_InvalidCommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bufRW := bufio.NewReadWriter(bufio.NewReader(&buf), bufio.NewWriter(&buf))

	// Write request with unsupported command (BIND)
	buf.Write([]byte{
		0x05,         // SOCKS version 5
		0x02,         // CMD: BIND (not supported)
		0x00,         // Reserved
		0x01,         // Address type: IPv4
		127, 0, 0, 1, // IP: 127.0.0.1
		0x00, 0x50, // Port: 80
	})

	_, err := handleRequest(bufRW)
	if err == nil {
		t.Error("Expected error with unsupported command, got nil")
	}

	bufRW.Flush()
	// Response should contain an error code
}

func TestHandleRequest_ValidConnect(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bufRW := bufio.NewReadWriter(bufio.NewReader(&buf), bufio.NewWriter(&buf))

	// Write a valid CONNECT request
	buf.Write([]byte{
		0x05,           // SOCKS version 5
		0x01,           // CMD: CONNECT
		0x00,           // Reserved
		0x01,           // Address type: IPv4
		192, 168, 1, 1, // IP: 192.168.1.1
		0x1F, 0x90, // Port: 8080
	})

	req, err := handleRequest(bufRW)
	if err != nil {
		t.Errorf("handleRequest() returned unexpected error: %v", err)
	}

	if req == nil {
		t.Fatal("Expected request object, got nil")
	}

	if req.Cmd != socks.CommandConnect {
		t.Errorf("Expected CONNECT command, got %v", req.Cmd)
	}

	if req.DstPort != 8080 {
		t.Errorf("Expected port 8080, got %d", req.DstPort)
	}
}

func TestHandleRequest_ValidAssociate(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bufRW := bufio.NewReadWriter(bufio.NewReader(&buf), bufio.NewWriter(&buf))

	// Write a valid ASSOCIATE request
	buf.Write([]byte{
		0x05,       // SOCKS version 5
		0x03,       // CMD: ASSOCIATE
		0x00,       // Reserved
		0x01,       // Address type: IPv4
		0, 0, 0, 0, // IP: 0.0.0.0 (client will send UDP to any address)
		0x00, 0x00, // Port: 0
	})

	req, err := handleRequest(bufRW)
	if err != nil {
		t.Errorf("handleRequest() returned unexpected error: %v", err)
	}

	if req == nil {
		t.Fatal("Expected request object, got nil")
	}

	if req.Cmd != socks.CommandAssociate {
		t.Errorf("Expected ASSOCIATE command, got %v", req.Cmd)
	}
}

func TestConfig_StringFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      Config
		expected string
	}{
		{
			name:     "standard config",
			cfg:      Config{LocalHost: "127.0.0.1", LocalPort: 1080},
			expected: "127.0.0.1:1080",
		},
		{
			name:     "all interfaces",
			cfg:      Config{LocalHost: "0.0.0.0", LocalPort: 9050},
			expected: "0.0.0.0:9050",
		},
		{
			name:     "localhost hostname",
			cfg:      Config{LocalHost: "localhost", LocalPort: 8080},
			expected: "localhost:8080",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.cfg.String()
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestNewUDPRelay_InvalidAddress verifies error handling for invalid addresses.
func TestNewUDPRelay_InvalidAddress(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	invalidAddr := "invalid::address::format"

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// Create a minimal valid request (ReadRequest would parse this from SOCKS protocol)
	// We use a buffer to simulate a SOCKS request with IPv4 address
	var buf bytes.Buffer
	buf.Write([]byte{
		0x05,           // SOCKS version 5
		0x01,           // CMD: CONNECT
		0x00,           // Reserved
		0x01,           // Address type: IPv4
		192, 168, 1, 1, // IP: 192.168.1.1
		0x00, 0x50, // Port: 80
	})
	req, err := socks.ReadRequest(&buf)
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}

	_, err = NewUDPRelay(ctx, invalidAddr, req, client, nil, nil)
	if err == nil {
		t.Error("NewUDPRelay() expected error with invalid address, got nil")
	}
}
