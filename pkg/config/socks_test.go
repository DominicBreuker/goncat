package config

import (
	"fmt"
	"testing"
)

func TestNewSocksCfg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		spec      string
		wantHost  string
		wantPort  int
		wantError bool
	}{
		{
			name:      "port only",
			spec:      "1080",
			wantHost:  "127.0.0.1",
			wantPort:  1080,
			wantError: false,
		},
		{
			name:      "host and port",
			spec:      "192.168.1.1:1080",
			wantHost:  "192.168.1.1",
			wantPort:  1080,
			wantError: false,
		},
		{
			name:      "localhost and port",
			spec:      "localhost:8080",
			wantHost:  "localhost",
			wantPort:  8080,
			wantError: false,
		},
		{
			name:      "invalid format - too many colons",
			spec:      "host:port:extra",
			wantHost:  "",
			wantPort:  0,
			wantError: true,
		},
		{
			name:      "invalid port",
			spec:      "localhost:abc",
			wantHost:  "",
			wantPort:  0,
			wantError: true,
		},
		{
			name:      "empty spec",
			spec:      "",
			wantHost:  "",
			wantPort:  0,
			wantError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := NewSocksCfg(tc.spec)

			if (cfg.parsingErr != nil) != tc.wantError {
				t.Errorf("NewSocksCfg(%q) parsingErr = %v, wantError %v", tc.spec, cfg.parsingErr, tc.wantError)
			}

			if !tc.wantError {
				if cfg.Host != tc.wantHost {
					t.Errorf("NewSocksCfg(%q) Host = %q, want %q", tc.spec, cfg.Host, tc.wantHost)
				}
				if cfg.Port != tc.wantPort {
					t.Errorf("NewSocksCfg(%q) Port = %d, want %d", tc.spec, cfg.Port, tc.wantPort)
				}
			}
		})
	}
}

func TestSocksCfg_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *SocksCfg
		want string
	}{
		{
			name: "valid config",
			cfg: &SocksCfg{
				Host: "localhost",
				Port: 1080,
				spec: "localhost:1080",
			},
			want: "localhost:1080",
		},
		{
			name: "config with parsing error",
			cfg: &SocksCfg{
				spec:       "invalid:spec:format",
				parsingErr: ErrInvalidFormat,
			},
			want: "invalid:spec:format",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.cfg.String(); got != tc.want {
				t.Errorf("SocksCfg.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSocksCfg_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *SocksCfg
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &SocksCfg{
				Host: "localhost",
				Port: 1080,
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			cfg: &SocksCfg{
				Host: "localhost",
				Port: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: &SocksCfg{
				Host: "localhost",
				Port: 65536,
			},
			wantErr: true,
		},
		{
			name: "valid port boundary - 1",
			cfg: &SocksCfg{
				Host: "localhost",
				Port: 1,
			},
			wantErr: false,
		},
		{
			name: "valid port boundary - 65535",
			cfg: &SocksCfg{
				Host: "localhost",
				Port: 65535,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := tc.cfg.validate()
			if (len(errs) > 0) != tc.wantErr {
				t.Errorf("SocksCfg.validate() errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}

var ErrInvalidFormat = fmt.Errorf("unexpected format")
