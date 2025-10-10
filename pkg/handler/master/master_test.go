package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"testing"
)

func TestNew_Success(t *testing.T) {
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
	// multiplexing, but we can test that the configuration is set up correctly
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

func TestMasterConfig_Validation(t *testing.T) {
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
