package portfwd

import (
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"errors"
	"net"
	"testing"
)

func TestConfig_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "standard config",
			cfg: Config{
				LocalHost:  "127.0.0.1",
				LocalPort:  8080,
				RemoteHost: "192.168.1.1",
				RemotePort: 9090,
			},
			want: "PortForwarding[127.0.0.1:8080 -> 192.168.1.1:9090]",
		},
		{
			name: "localhost to localhost",
			cfg: Config{
				LocalHost:  "localhost",
				LocalPort:  3000,
				RemoteHost: "localhost",
				RemotePort: 4000,
			},
			want: "PortForwarding[localhost:3000 -> localhost:4000]",
		},
		{
			name: "wildcard address",
			cfg: Config{
				LocalHost:  "*",
				LocalPort:  1234,
				RemoteHost: "example.com",
				RemotePort: 443,
			},
			want: "PortForwarding[*:1234 -> example.com:443]",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.cfg.String()
			if got != tc.want {
				t.Errorf("Config.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := Config{
		LocalHost:  "127.0.0.1",
		LocalPort:  8080,
		RemoteHost: "192.168.1.1",
		RemotePort: 9090,
	}

	srv := NewServer(ctx, cfg, nil)

	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}
	if srv.ctx != ctx {
		t.Error("NewServer() did not set context correctly")
	}
	if srv.cfg.LocalHost != cfg.LocalHost {
		t.Errorf("NewServer() LocalHost = %q, want %q", srv.cfg.LocalHost, cfg.LocalHost)
	}
	if srv.cfg.LocalPort != cfg.LocalPort {
		t.Errorf("NewServer() LocalPort = %d, want %d", srv.cfg.LocalPort, cfg.LocalPort)
	}
}

type fakeServerControlSession struct {
	channelFn func(m msg.Message) (net.Conn, error)
}

func (f *fakeServerControlSession) SendAndGetOneChannel(m msg.Message) (net.Conn, error) {
	if f.channelFn != nil {
		return f.channelFn(m)
	}
	return nil, nil
}

func TestNewServer_AllFields(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		LocalHost:  "0.0.0.0",
		LocalPort:  12345,
		RemoteHost: "10.0.0.1",
		RemotePort: 54321,
	}

	sessCtl := &fakeServerControlSession{}
	srv := NewServer(ctx, cfg, sessCtl)

	if srv.ctx != ctx {
		t.Error("Server context not set correctly")
	}
	if srv.cfg != cfg {
		t.Error("Server config not set correctly")
	}
	if srv.sessCtl != sessCtl {
		t.Error("Server control session not set correctly")
	}
}

// TestServer_HandlePortForwardingConn verifies the connection handling logic.
func TestServer_HandlePortForwardingConn_SendAndGetError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := Config{
		LocalHost:  "127.0.0.1",
		LocalPort:  0,
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	// Create a fake session that returns an error
	sessCtl := &fakeServerControlSession{
		channelFn: func(m msg.Message) (net.Conn, error) {
			return nil, errors.New("test error")
		},
	}

	srv := NewServer(ctx, cfg, sessCtl)

	// Create a fake local connection
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	err := srv.handlePortForwardingConn(client)
	if err == nil {
		t.Error("handlePortForwardingConn() expected error, got nil")
	}
}

// TestServer_Handle_InvalidAddress verifies error handling for invalid addresses.
func TestServer_Handle_InvalidAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := Config{
		LocalHost:  "invalid::address",
		LocalPort:  8080,
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	sessCtl := &fakeServerControlSession{}
	srv := NewServer(ctx, cfg, sessCtl)

	err := srv.Handle()
	if err == nil {
		t.Error("Handle() expected error with invalid address, got nil")
	}
}
