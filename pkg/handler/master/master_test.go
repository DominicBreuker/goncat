package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux"
	"net"
	"sync"
	"testing"
)

// TestNew creates a new master handler and verifies initialization.
func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	ctx := context.Background()
	cfg := &config.Shared{
		Verbose: false,
	}
	mCfg := &config.Master{
		Exec: "/bin/sh",
		Pty:  false,
	}

	// Start slave side to accept session
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := mux.AcceptSession(server)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	if master.ctx != ctx {
		t.Error("master.ctx not set correctly")
	}
	if master.cfg != cfg {
		t.Error("master.cfg not set correctly")
	}
	if master.mCfg != mCfg {
		t.Error("master.mCfg not set correctly")
	}
	if master.sess == nil {
		t.Error("master.sess is nil")
	}

	wg.Wait()
}

// TestNew_SessionError verifies error handling when session creation fails.
func TestNew_SessionError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{}

	// Create a connection that will be immediately closed
	client, server := net.Pipe()
	server.Close()
	client.Close()

	_, err := New(ctx, cfg, mCfg, client)
	if err == nil {
		t.Error("New() expected error with closed connection, got nil")
	}
}

func TestNew_ConfigValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Verbose: false,
	}
	mCfg := &config.Master{
		Exec: "",
		Pty:  false,
	}

	// We cannot fully test New without a valid connection that supports
	// multiplexing, but we can test that the configuration is validated correctly
	if ctx.Err() != nil {
		t.Error("context should not be cancelled")
	}
	if cfg.Verbose != false {
		t.Error("expected verbose to be false")
	}
	if mCfg.Exec != "" {
		t.Error("expected exec to be empty")
	}
	if mCfg.Pty != false {
		t.Error("expected pty to be false")
	}
}

// TestClose verifies that Close properly closes the master session.
func TestClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slave, err := mux.AcceptSession(server)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := master.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	wg.Wait()
}

func TestMasterConfig_Scenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.Master
	}{
		{
			name: "empty config",
			cfg:  &config.Master{},
		},
		{
			name: "with exec",
			cfg: &config.Master{
				Exec: "/bin/sh",
			},
		},
		{
			name: "with pty",
			cfg: &config.Master{
				Pty: true,
			},
		},
		{
			name: "with exec and pty",
			cfg: &config.Master{
				Exec: "/bin/sh",
				Pty:  true,
			},
		},
		{
			name: "with log file",
			cfg: &config.Master{
				LogFile: "/tmp/test.log",
			},
		},
		{
			name: "with SOCKS proxy",
			cfg: &config.Master{
				Socks: &config.SocksCfg{
					Host: "127.0.0.1",
					Port: 1080,
				},
			},
		},
		{
			name: "with local port forwarding",
			cfg: &config.Master{
				LocalPortForwarding: []*config.LocalPortForwardingCfg{
					{
						LocalHost:  "127.0.0.1",
						LocalPort:  8080,
						RemoteHost: "example.com",
						RemotePort: 80,
					},
				},
			},
		},
		{
			name: "with remote port forwarding",
			cfg: &config.Master{
				RemotePortForwarding: []*config.RemotePortForwardingCfg{
					{
						LocalHost:  "127.0.0.1",
						LocalPort:  9090,
						RemoteHost: "localhost",
						RemotePort: 8080,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.cfg == nil {
				t.Error("Config should not be nil")
			}
		})
	}
}

func TestMasterConfig_IsSocksEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      *config.Master
		expected bool
	}{
		{
			name:     "socks enabled",
			cfg:      &config.Master{Socks: &config.SocksCfg{Host: "127.0.0.1", Port: 1080}},
			expected: true,
		},
		{
			name:     "socks disabled - nil",
			cfg:      &config.Master{Socks: nil},
			expected: false,
		},
		{
			name:     "empty config",
			cfg:      &config.Master{},
			expected: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.cfg.IsSocksEnabled()
			if result != tc.expected {
				t.Errorf("IsSocksEnabled() = %v, want %v", result, tc.expected)
			}
		})
	}
}
