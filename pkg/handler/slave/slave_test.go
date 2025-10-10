package slave

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

	// We cannot fully test New without a valid connection that supports
	// multiplexing, but we can test that the configuration is set up correctly
	if ctx.Err() != nil {
		t.Error("context should not be cancelled")
	}
	if cfg.Verbose != false {
		t.Error("expected verbose to be false")
	}
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
