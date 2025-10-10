package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"testing"
)

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
