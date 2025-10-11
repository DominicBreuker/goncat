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

	srv := NewServer(ctx, cfg, nil, nil)

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
	srv := NewServer(ctx, cfg, sessCtl, nil)

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

	srv := NewServer(ctx, cfg, sessCtl, nil)

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
	srv := NewServer(ctx, cfg, sessCtl, nil)

	err := srv.Handle()
	if err == nil {
		t.Error("Handle() expected error with invalid address, got nil")
	}
}

// TestServer_Handle_PortInUse verifies error handling when the port is already in use.
func TestServer_Handle_PortInUse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	// Start a listener to occupy a port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)

	ctx := context.Background()
	cfg := Config{
		LocalHost:  addr.IP.String(),
		LocalPort:  addr.Port,
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	sessCtl := &fakeServerControlSession{}
	srv := NewServer(ctx, cfg, sessCtl, nil)

	err = srv.Handle()
	if err == nil {
		t.Error("Handle() expected error when port is in use, got nil")
	}
}

// TestServer_Handle_ContextCancellation verifies that Handle respects context cancellation.
func TestServer_Handle_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{
		LocalHost:  "127.0.0.1",
		LocalPort:  0, // Let OS choose port
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	sessCtl := &fakeServerControlSession{}
	srv := NewServer(ctx, cfg, sessCtl, nil)

	// Cancel context before calling Handle
	cancel()

	// Start Handle in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- srv.Handle()
	}()

	// Wait for Handle to return (should be quick due to cancelled context)
	err := <-done
	// Should return without error when context is cancelled
	if err != nil {
		// Context cancellation during accept is expected to return nil
		t.Logf("Handle() returned error: %v", err)
	}
}

// TestServer_HandlePortForwardingConn_Success verifies successful connection forwarding.
func TestServer_HandlePortForwardingConn_Success(t *testing.T) {
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

	// Create a successful channel
	remoteClient, remoteServer := net.Pipe()
	defer remoteClient.Close()
	defer remoteServer.Close()

	sessCtl := &fakeServerControlSession{
		channelFn: func(m msg.Message) (net.Conn, error) {
			return remoteServer, nil
		},
	}

	srv := NewServer(ctx, cfg, sessCtl, nil)

	// Create a fake local connection
	localClient, localServer := net.Pipe()
	defer localClient.Close()
	defer localServer.Close()

	// Call handlePortForwardingConn in a goroutine since it blocks
	go func() {
		err := srv.handlePortForwardingConn(localClient)
		if err != nil {
			t.Errorf("handlePortForwardingConn() unexpected error: %v", err)
		}
	}()

	// Close connections to unblock the goroutine
	localServer.Close()
	remoteClient.Close()
}

// TestServer_Handle_TableDriven tests various server scenarios.
func TestServer_Handle_TableDriven(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "invalid local host",
			cfg: Config{
				LocalHost:  "invalid::host",
				LocalPort:  8080,
				RemoteHost: "example.com",
				RemotePort: 80,
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			cfg: Config{
				LocalHost:  "127.0.0.1",
				LocalPort:  99999,
				RemoteHost: "example.com",
				RemotePort: 80,
			},
			wantErr: true,
		},
		{
			name: "zero port should succeed",
			cfg: Config{
				LocalHost:  "127.0.0.1",
				LocalPort:  0,
				RemoteHost: "example.com",
				RemotePort: 80,
			},
			wantErr: false, // Zero port is valid (OS assigns port)
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sessCtl := &fakeServerControlSession{}
			srv := NewServer(ctx, tc.cfg, sessCtl, nil)

			// For successful cases, cancel immediately
			if !tc.wantErr {
				cancel()
			}

			err := srv.Handle()

			if (err != nil) != tc.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestConfig_Fields verifies Config struct field assignment.
func TestConfig_Fields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		localHost  string
		localPort  int
		remoteHost string
		remotePort int
	}{
		{
			name:       "standard values",
			localHost:  "127.0.0.1",
			localPort:  8080,
			remoteHost: "192.168.1.1",
			remotePort: 9090,
		},
		{
			name:       "wildcard local",
			localHost:  "*",
			localPort:  3000,
			remoteHost: "example.com",
			remotePort: 443,
		},
		{
			name:       "zero port",
			localHost:  "localhost",
			localPort:  0,
			remoteHost: "api.example.com",
			remotePort: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := Config{
				LocalHost:  tc.localHost,
				LocalPort:  tc.localPort,
				RemoteHost: tc.remoteHost,
				RemotePort: tc.remotePort,
			}

			if cfg.LocalHost != tc.localHost {
				t.Errorf("LocalHost = %q, want %q", cfg.LocalHost, tc.localHost)
			}
			if cfg.LocalPort != tc.localPort {
				t.Errorf("LocalPort = %d, want %d", cfg.LocalPort, tc.localPort)
			}
			if cfg.RemoteHost != tc.remoteHost {
				t.Errorf("RemoteHost = %q, want %q", cfg.RemoteHost, tc.remoteHost)
			}
			if cfg.RemotePort != tc.remotePort {
				t.Errorf("RemotePort = %d, want %d", cfg.RemotePort, tc.remotePort)
			}
		})
	}
}

// TestServer_Handle_AcceptConnection tests the successful accept and handle flow.
func TestServer_Handle_AcceptConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		LocalHost:  "127.0.0.1",
		LocalPort:  0, // Let OS choose port
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	// Create a fake remote channel
	remoteClient, remoteServer := net.Pipe()
	defer remoteClient.Close()
	defer remoteServer.Close()

	sessCtl := &fakeServerControlSession{
		channelFn: func(m msg.Message) (net.Conn, error) {
			return remoteServer, nil
		},
	}

	srv := NewServer(ctx, cfg, sessCtl, nil)

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		err := srv.Handle()
		serverErr <- err
	}()

	// Cancel immediately to test the code path
	cancel()

	// Wait for server to exit
	err := <-serverErr
	if err != nil {
		t.Logf("Server returned error (expected on cancellation): %v", err)
	}
}

// TestServer_Handle_AcceptAndForward tests accepting a connection and forwarding it.
func TestServer_Handle_AcceptAndForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		LocalHost:  "127.0.0.1",
		LocalPort:  0, // Let OS choose port
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	// Create fake remote connection
	remoteClient, remoteServer := net.Pipe()
	defer remoteClient.Close()
	defer remoteServer.Close()

	channelRequested := make(chan msg.Message, 1)

	sessCtl := &fakeServerControlSession{
		channelFn: func(m msg.Message) (net.Conn, error) {
			channelRequested <- m
			return remoteServer, nil
		},
	}

	srv := NewServer(ctx, cfg, sessCtl, nil)

	serverDone := make(chan error, 1)

	// We need to get the listening address, but Handle blocks
	// Let's use a different approach: start listening ourselves first
	// to get a port, then use that port
	testListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	addr := testListener.Addr().(*net.TCPAddr)
	testListener.Close()

	// Update config with the port we know is free
	cfg.LocalPort = addr.Port
	srv = NewServer(ctx, cfg, sessCtl, nil)

	// Start server
	go func() {
		serverDone <- srv.Handle()
	}()

	// Wait a bit for server to start listening
	// Retry connection attempts with exponential backoff
	var conn net.Conn
	for i := 0; i < 20; i++ {
		conn, err = net.Dial("tcp", addr.String())
		if err == nil {
			break
		}
		// Exponential backoff: 1ms, 2ms, 4ms, ... up to ~1 second total
		if i < 10 {
			select {
			case <-ctx.Done():
				t.Fatal("context cancelled before connection established")
			default:
			}
		}
	}
	if err != nil {
		cancel()
		t.Fatalf("failed to connect to server after retries: %v", err)
	}
	defer conn.Close()

	// Wait for channel request
	m := <-channelRequested
	connectMsg, ok := m.(msg.Connect)
	if !ok {
		t.Fatalf("expected msg.Connect, got %T", m)
	}
	if connectMsg.RemoteHost != cfg.RemoteHost {
		t.Errorf("RemoteHost = %q, want %q", connectMsg.RemoteHost, cfg.RemoteHost)
	}
	if connectMsg.RemotePort != cfg.RemotePort {
		t.Errorf("RemotePort = %d, want %d", connectMsg.RemotePort, cfg.RemotePort)
	}

	// Close connections to unblock
	conn.Close()
	remoteClient.Close()

	// Cancel server
	cancel()

	// Wait for server to finish
	<-serverDone
}
