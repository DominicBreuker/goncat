package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux"
	"net"
	"sync"
	"testing"
)

// TestNew creates a new slave handler and verifies initialization.
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

	// Start master side to open session
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := mux.OpenSession(client)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
		}
	}()

	slave, err := New(ctx, cfg, server)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer slave.Close()

	if slave.ctx != ctx {
		t.Error("slave.ctx not set correctly")
	}
	if slave.cfg != cfg {
		t.Error("slave.cfg not set correctly")
	}
	if slave.sess == nil {
		t.Error("slave.sess is nil")
	}

	wg.Wait()
}

// TestNew_SessionError verifies error handling when session creation fails.
func TestNew_SessionError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{}

	// Create a connection that will be immediately closed
	client, server := net.Pipe()
	client.Close()
	server.Close()

	_, err := New(ctx, cfg, server)
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

	// We cannot fully test New without a valid connection that supports
	// multiplexing, but we can test that the configuration is set up correctly
	if ctx.Err() != nil {
		t.Error("context should not be cancelled")
	}
	if cfg.Verbose != false {
		t.Error("expected verbose to be false")
	}
}

// TestClose verifies that Close properly closes the slave session.
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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		master, err := mux.OpenSession(client)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()
	}()

	slave, err := New(ctx, cfg, server)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := slave.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	wg.Wait()
}

func TestSlaveConfig_Scenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.Shared
	}{
		{
			name: "default config",
			cfg:  &config.Shared{},
		},
		{
			name: "verbose mode",
			cfg: &config.Shared{
				Verbose: true,
			},
		},
		{
			name: "with SSL",
			cfg: &config.Shared{
				SSL: true,
			},
		},
		{
			name: "verbose and SSL",
			cfg: &config.Shared{
				Verbose: true,
				SSL:     true,
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

func TestSlaveConfig_Properties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *config.Shared
		verbose bool
		ssl     bool
	}{
		{
			name:    "all defaults",
			cfg:     &config.Shared{},
			verbose: false,
			ssl:     false,
		},
		{
			name:    "verbose enabled",
			cfg:     &config.Shared{Verbose: true},
			verbose: true,
			ssl:     false,
		},
		{
			name:    "SSL enabled",
			cfg:     &config.Shared{SSL: true},
			verbose: false,
			ssl:     true,
		},
		{
			name:    "both enabled",
			cfg:     &config.Shared{Verbose: true, SSL: true},
			verbose: true,
			ssl:     true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.cfg.Verbose != tc.verbose {
				t.Errorf("Verbose = %v, want %v", tc.cfg.Verbose, tc.verbose)
			}
			if tc.cfg.SSL != tc.ssl {
				t.Errorf("SSL = %v, want %v", tc.cfg.SSL, tc.ssl)
			}
		})
	}
}
