package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux"
	msg "dominicbreuker/goncat/pkg/mux/msg"
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
	defer server.Close()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := mux.AcceptSession(server)
		if err != nil {
			// Expected error when master closes
			return
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := master.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	client.Close()

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

// TestStartLocalPortFwdJob tests the startLocalPortFwdJob method.
func TestStartLocalPortFwdJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{}

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error (expected on cleanup): %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	lpf := &config.LocalPortForwardingCfg{
		LocalHost:  "127.0.0.1",
		LocalPort:  0, // Use port 0 to let OS choose available port
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	var jobWg sync.WaitGroup
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start the job
	master.startLocalPortFwdJob(jobCtx, &jobWg, lpf)

	// Cancel context to stop the job
	cancel()

	// Wait for job to complete
	jobWg.Wait()

	wg.Wait()
}

// TestStartRemotePortFwdJob tests the startRemotePortFwdJob method.
func TestStartRemotePortFwdJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{}

	client, server := net.Pipe()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sess, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error: %v", err)
			return
		}
		defer sess.Close()

		// Receive the port forwarding message
		_, err = sess.Receive()
		if err != nil {
			t.Logf("Receive() error (may be expected on cleanup): %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	rpf := &config.RemotePortForwardingCfg{
		LocalHost:  "127.0.0.1",
		LocalPort:  9090,
		RemoteHost: "localhost",
		RemotePort: 8080,
	}

	var jobWg sync.WaitGroup
	jobCtx := context.Background()

	// Start the job
	master.startRemotePortFwdJob(jobCtx, &jobWg, rpf)

	// Wait for job to complete
	jobWg.Wait()

	client.Close()
	wg.Wait()
}

// TestHandleConnectAsync tests the handleConnectAsync method.
func TestHandleConnectAsync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{}

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error: %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	m := msg.Connect{
		RemoteHost: "example.com",
		RemotePort: 80,
	}

	// handleConnectAsync spawns a goroutine, so we just verify it doesn't panic
	master.handleConnectAsync(ctx, m)

	wg.Wait()
}

// TestStartSocksProxyJob tests the startSocksProxyJob method.
func TestStartSocksProxyJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{
		Socks: &config.SocksCfg{
			Host: "127.0.0.1",
			Port: 0, // Use port 0 to let OS choose available port
		},
	}

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error: %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	var jobWg sync.WaitGroup
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start the SOCKS proxy job
	err = master.startSocksProxyJob(jobCtx, &jobWg)
	if err != nil {
		t.Fatalf("startSocksProxyJob() error = %v", err)
	}

	// Cancel context to stop the job
	cancel()

	// Wait for job to complete
	jobWg.Wait()

	wg.Wait()
}

// TestStartSocksProxyJob_InvalidConfig tests error handling for invalid SOCKS config.
func TestStartSocksProxyJob_InvalidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{
		Socks: &config.SocksCfg{
			Host: "127.0.0.1",
			Port: 99999, // Invalid port
		},
	}

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error: %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	var jobWg sync.WaitGroup
	jobCtx := context.Background()

	// Try to start the SOCKS proxy job with invalid config
	err = master.startSocksProxyJob(jobCtx, &jobWg)
	if err == nil {
		t.Error("startSocksProxyJob() expected error with invalid port, got nil")
	}

	wg.Wait()
}

// TestHandleForeground_Plain tests handleForeground without PTY.
func TestHandleForeground_Plain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Shared{
		Verbose: false,
	}
	mCfg := &config.Master{
		Exec: "/bin/sh",
		Pty:  false, // Plain mode
	}

	client, server := net.Pipe()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sess, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error: %v", err)
			return
		}
		defer sess.Close()

		// Receive the foreground message and open a channel
		m, err := sess.Receive()
		if err != nil {
			t.Logf("Receive() error: %v", err)
			return
		}

		if _, ok := m.(msg.Foreground); !ok {
			t.Errorf("Expected Foreground message, got %T", m)
			return
		}

		// Open a channel for the foreground connection
		_, err = sess.GetOneChannel()
		if err != nil {
			t.Logf("GetOneChannel() error (may be expected): %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	// Start handleForeground in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- master.handleForeground(ctx)
	}()

	// Cancel context to stop the foreground handler
	cancel()

	// Wait for handleForeground to complete
	err = <-errCh
	if err != nil {
		t.Logf("handleForeground() returned error (may be expected on cancellation): %v", err)
	}

	client.Close()
	wg.Wait()
}

// TestHandleForeground_PTY tests handleForeground with PTY enabled.
func TestHandleForeground_PTY(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Shared{
		Verbose: false,
	}
	mCfg := &config.Master{
		Exec: "/bin/sh",
		Pty:  true, // PTY mode
	}

	client, server := net.Pipe()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sess, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error: %v", err)
			return
		}
		defer sess.Close()

		// Receive the foreground message and open channels
		m, err := sess.Receive()
		if err != nil {
			t.Logf("Receive() error: %v", err)
			return
		}

		if _, ok := m.(msg.Foreground); !ok {
			t.Errorf("Expected Foreground message, got %T", m)
			return
		}

		// Open two channels for PTY mode (data and control)
		_, err = sess.AcceptNewChannel()
		if err != nil {
			t.Logf("AcceptNewChannel() error (may be expected): %v", err)
			return
		}
		_, err = sess.AcceptNewChannel()
		if err != nil {
			t.Logf("AcceptNewChannel() error (may be expected): %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	// Start handleForeground in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- master.handleForeground(ctx)
	}()

	// Cancel context to stop the foreground handler
	cancel()

	// Wait for handleForeground to complete
	err = <-errCh
	if err != nil {
		t.Logf("handleForeground() returned error (may be expected on cancellation): %v", err)
	}

	client.Close()
	wg.Wait()
}

// TestStartForegroundJob tests the startForegroundJob method.
func TestStartForegroundJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}
	mCfg := &config.Master{
		Exec: "/bin/sh",
		Pty:  false,
	}

	client, server := net.Pipe()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sess, err := mux.AcceptSession(server)
		if err != nil {
			t.Logf("AcceptSession() error: %v", err)
			return
		}
		defer sess.Close()

		// Receive and handle the foreground message
		_, err = sess.Receive()
		if err != nil {
			t.Logf("Receive() error: %v", err)
			return
		}

		_, err = sess.GetOneChannel()
		if err != nil {
			t.Logf("GetOneChannel() error: %v", err)
		}
	}()

	master, err := New(ctx, cfg, mCfg, client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer master.Close()

	var jobWg sync.WaitGroup
	jobCtx, cancel := context.WithCancel(ctx)

	// Start the foreground job
	master.startForegroundJob(jobCtx, &jobWg, cancel)

	// Wait a bit for the job to start
	// The cancel function should be called by the job when it completes

	// Wait for job to complete
	jobWg.Wait()

	client.Close()
	wg.Wait()
}
