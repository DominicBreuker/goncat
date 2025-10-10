package portfwd

import (
	"context"
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

func TestNewClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	m := msg.Connect{
		RemoteHost: "example.com",
		RemotePort: 443,
	}

	client := NewClient(ctx, m, nil)

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
	client := NewClient(ctx, m, sessCtl)

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

			client := NewClient(ctx, m, nil)

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
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
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

	client := NewClient(ctx, m, sessCtl)
	err := client.Handle()

	if err == nil {
		t.Error("Handle() expected error when GetOneChannel fails, got nil")
	}
}

// TestClient_Handle_InvalidAddress verifies error handling for invalid destination addresses.
func TestClient_Handle_InvalidAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
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

	client := NewClient(ctx, m, sessCtl)
	err := client.Handle()

	if err == nil {
		t.Error("Handle() expected error with invalid address, got nil")
	}
}

// TestClient_Handle_DialError verifies error handling when dialing the destination fails.
func TestClient_Handle_DialError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	// Use localhost with a port that's very unlikely to be listening
	// This will fail quickly with "connection refused" rather than timeout
	m := msg.Connect{
		RemoteHost: "127.0.0.1",
		RemotePort: 1, // Port 1 requires root, should be refused
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

	client := NewClient(ctx, m, sessCtl)
	err := client.Handle()

	if err == nil {
		t.Error("Handle() expected error when dial fails, got nil")
	}
}

// TestClient_Handle_TableDriven tests various error scenarios using table-driven approach.
func TestClient_Handle_TableDriven(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	tests := []struct {
		name      string
		msg       msg.Connect
		channelFn func() (net.Conn, error)
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
			wantErr: true,
		},
		{
			name: "connection refused",
			msg: msg.Connect{
				RemoteHost: "127.0.0.1",
				RemotePort: 1, // Port 1 should be refused
			},
			channelFn: func() (net.Conn, error) {
				client, server := net.Pipe()
				go func() { client.Close() }()
				return server, nil
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

			client := NewClient(ctx, tc.msg, sessCtl)
			err := client.Handle()

			if (err != nil) != tc.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestClient_ContextCancellation verifies that the client respects context cancellation.
func TestClient_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
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

	client := NewClient(ctx, m, sessCtl)

	// Cancel context immediately to test cancellation path
	cancel()

	// The Handle should eventually respect the cancelled context
	// This test mainly verifies no panic occurs
	_ = client.Handle()
}
