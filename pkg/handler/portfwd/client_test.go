package portfwd

import (
	"context"
	mocks_tcp "dominicbreuker/goncat/mocks/tcp"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"errors"
	"net"
	"testing"
)

type fakeClientControlSession struct {
	channelFn func() (net.Conn, error)
}

func (f *fakeClientControlSession) GetOneChannel() (net.Conn, error) {
	if f.channelFn != nil {
		return f.channelFn()
	}
	return nil, nil
}

func (f *fakeClientControlSession) GetOneChannelContext(ctx context.Context) (net.Conn, error) {
	return f.GetOneChannel()
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	m := msg.Connect{
		RemoteHost: "example.com",
		RemotePort: 443,
	}

	client := NewClient(ctx, m, nil, log.NewLogger(false), nil)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.ctx != ctx {
		t.Error("NewClient() did not set context correctly")
	}
	if client.m.RemoteHost != m.RemoteHost {
		t.Errorf("NewClient() RemoteHost = %q, want %q", client.m.RemoteHost, m.RemoteHost)
	}
	if client.m.RemotePort != m.RemotePort {
		t.Errorf("NewClient() RemotePort = %d, want %d", client.m.RemotePort, m.RemotePort)
	}
}

func TestNewClient_AllFields(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := msg.Connect{
		RemoteHost: "192.168.1.100",
		RemotePort: 8080,
	}

	sessCtl := &fakeClientControlSession{}
	client := NewClient(ctx, m, sessCtl, log.NewLogger(false), nil)

	if client.ctx != ctx {
		t.Error("Client context not set correctly")
	}
	if client.m != m {
		t.Error("Client message not set correctly")
	}
	if client.sessCtl != sessCtl {
		t.Error("Client control session not set correctly")
	}
}

// TestNewClient_DifferentPorts verifies client creation with various port values.
func TestNewClient_DifferentPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		port int
	}{
		{"standard http", "example.com", 80},
		{"https", "secure.example.com", 443},
		{"custom port", "api.example.com", 8080},
		{"high port", "test.example.com", 65535},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			m := msg.Connect{
				RemoteHost: tc.host,
				RemotePort: tc.port,
			}

			client := NewClient(ctx, m, nil, log.NewLogger(false), nil)

			if client == nil {
				t.Fatal("NewClient() returned nil")
			}
			if client.m.RemoteHost != tc.host {
				t.Errorf("RemoteHost = %q, want %q", client.m.RemoteHost, tc.host)
			}
			if client.m.RemotePort != tc.port {
				t.Errorf("RemotePort = %d, want %d", client.m.RemotePort, tc.port)
			}
		})
	}
}

// TestClient_Handle_GetChannelError verifies error handling when GetOneChannel fails.
func TestClient_Handle_GetChannelError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	m := msg.Connect{
		RemoteHost: "example.com",
		RemotePort: 443,
	}

	expectedErr := errors.New("channel error")
	sessCtl := &fakeClientControlSession{
		channelFn: func() (net.Conn, error) {
			return nil, expectedErr
		},
	}

	client := NewClient(ctx, m, sessCtl, log.NewLogger(false), nil)
	err := client.Handle()

	if err == nil {
		t.Error("Handle() expected error when GetOneChannel fails, got nil")
	}
}

// TestClient_Handle_InvalidAddress verifies error handling for invalid destination addresses.
func TestClient_Handle_InvalidAddress(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	m := msg.Connect{
		RemoteHost: "invalid::host::name",
		RemotePort: 80,
	}

	// Create a fake channel
	client1, server1 := net.Pipe()
	defer client1.Close()
	defer server1.Close()

	sessCtl := &fakeClientControlSession{
		channelFn: func() (net.Conn, error) {
			return server1, nil
		},
	}

	client := NewClient(ctx, m, sessCtl, log.NewLogger(false), nil)
	err := client.Handle()

	if err == nil {
		t.Error("Handle() expected error with invalid address, got nil")
	}
}

// TestClient_Handle_DialError verifies error handling when dialing the destination fails.
func TestClient_Handle_DialError(t *testing.T) {
	t.Parallel()

	// Use mock network with no listener to simulate connection refused
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer: mockNet.DialTCPContext,
	}

	m := msg.Connect{
		RemoteHost: "127.0.0.1",
		RemotePort: 12345, // No listener on this address
	}

	// Create a fake channel
	client1, server1 := net.Pipe()
	defer client1.Close()
	defer server1.Close()

	ctx := context.Background()
	sessCtl := &fakeClientControlSession{
		channelFn: func() (net.Conn, error) {
			return server1, nil
		},
	}

	client := NewClient(ctx, m, sessCtl, log.NewLogger(false), deps)
	err := client.Handle()

	if err == nil {
		t.Error("Handle() expected error when dial fails, got nil")
	}
}

// TestClient_Handle_TableDriven tests various error scenarios using table-driven approach.
func TestClient_Handle_TableDriven(t *testing.T) {
	t.Parallel()

	// Create mock network for connection refused test
	mockNet := mocks_tcp.NewMockTCPNetwork()

	tests := []struct {
		name      string
		msg       msg.Connect
		channelFn func() (net.Conn, error)
		deps      *config.Dependencies
		wantErr   bool
	}{
		{
			name: "GetOneChannel error",
			msg: msg.Connect{
				RemoteHost: "example.com",
				RemotePort: 80,
			},
			channelFn: func() (net.Conn, error) {
				return nil, errors.New("channel error")
			},
			deps:    nil,
			wantErr: true,
		},
		{
			name: "invalid host format",
			msg: msg.Connect{
				RemoteHost: "invalid::host",
				RemotePort: 80,
			},
			channelFn: func() (net.Conn, error) {
				client, server := net.Pipe()
				// Close client so test cleans up properly
				go func() { client.Close() }()
				return server, nil
			},
			deps:    nil,
			wantErr: true,
		},
		{
			name: "connection refused",
			msg: msg.Connect{
				RemoteHost: "127.0.0.1",
				RemotePort: 54321, // No listener on this address
			},
			channelFn: func() (net.Conn, error) {
				client, server := net.Pipe()
				go func() { client.Close() }()
				return server, nil
			},
			deps: &config.Dependencies{
				TCPDialer: mockNet.DialTCPContext,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			sessCtl := &fakeClientControlSession{
				channelFn: tc.channelFn,
			}

			client := NewClient(ctx, tc.msg, sessCtl, log.NewLogger(false), tc.deps)
			err := client.Handle()

			if (err != nil) != tc.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestClient_ContextCancellation verifies that the client respects context cancellation.
func TestClient_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	m := msg.Connect{
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	// Create channels that won't close on their own
	remoteClient, remoteServer := net.Pipe()
	defer remoteClient.Close()
	defer remoteServer.Close()

	sessCtl := &fakeClientControlSession{
		channelFn: func() (net.Conn, error) {
			return remoteServer, nil
		},
	}

	client := NewClient(ctx, m, sessCtl, log.NewLogger(false), nil)

	// Cancel context immediately to test cancellation path
	cancel()

	// The Handle should eventually respect the cancelled context
	// This test mainly verifies no panic occurs
	_ = client.Handle()
}

// TestClient_Handle_SuccessfulConnection tests the successful connection path.
func TestClient_Handle_SuccessfulConnection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Use mock TCP network
	mockNet := mocks_tcp.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCPContext,
		TCPListener: mockNet.ListenTCP,
	}

	// Start a listener on the mock network
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12350")
	listener, err := mockNet.ListenTCP("tcp", tcpAddr)
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	defer listener.Close()

	m := msg.Connect{
		RemoteHost: "127.0.0.1",
		RemotePort: 12350,
	}

	// Create a fake remote channel
	remoteClient, remoteServer := net.Pipe()
	defer remoteClient.Close()

	sessCtl := &fakeClientControlSession{
		channelFn: func() (net.Conn, error) {
			return remoteServer, nil
		},
	}

	// Accept connection in background
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	client := NewClient(ctx, m, sessCtl, log.NewLogger(false), deps)

	// Run Handle in goroutine since it blocks
	done := make(chan error, 1)
	go func() {
		done <- client.Handle()
	}()

	// Wait for connection to be accepted
	var localConn net.Conn
	select {
	case localConn = <-accepted:
		defer localConn.Close()
	case <-done:
		// Handle returned early, check for error
		err := <-done
		if err != nil {
			t.Logf("Handle() returned error: %v", err)
		}
		return
	}

	// Close connections to trigger completion
	remoteClient.Close()
	localConn.Close()
	remoteServer.Close()

	// Wait for Handle to complete
	err = <-done
	if err != nil {
		t.Logf("Handle() returned error: %v", err)
	}
}
