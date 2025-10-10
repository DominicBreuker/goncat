package config

import (
	"testing"
)

func TestNewLocalPortForwardingCfg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           string
		wantLocalHost  string
		wantLocalPort  int
		wantRemoteHost string
		wantRemotePort int
		wantError      bool
	}{
		{
			name:           "three parts",
			spec:           "8080:remote.host:80",
			wantLocalHost:  "",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "four parts",
			spec:           "localhost:8080:remote.host:80",
			wantLocalHost:  "localhost",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "invalid format - too few parts",
			spec:           "8080:80",
			wantLocalHost:  "",
			wantLocalPort:  0,
			wantRemoteHost: "",
			wantRemotePort: 0,
			wantError:      true,
		},
		{
			name:           "invalid format - too many parts",
			spec:           "a:b:c:d:e",
			wantLocalHost:  "",
			wantLocalPort:  0,
			wantRemoteHost: "",
			wantRemotePort: 0,
			wantError:      true,
		},
		{
			name:           "invalid local port",
			spec:           "abc:remote.host:80",
			wantLocalHost:  "",
			wantLocalPort:  0,
			wantRemoteHost: "",
			wantRemotePort: 0,
			wantError:      true,
		},
		{
			name:           "invalid remote port",
			spec:           "8080:remote.host:abc",
			wantLocalHost:  "",
			wantLocalPort:  0,
			wantRemoteHost: "",
			wantRemotePort: 0,
			wantError:      true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := newLocalPortForwardingCfg(tc.spec)

			if (cfg.parsingErr != nil) != tc.wantError {
				t.Errorf("newLocalPortForwardingCfg(%q) parsingErr = %v, wantError %v", tc.spec, cfg.parsingErr, tc.wantError)
			}

			if !tc.wantError {
				if cfg.LocalHost != tc.wantLocalHost {
					t.Errorf("LocalHost = %q, want %q", cfg.LocalHost, tc.wantLocalHost)
				}
				if cfg.LocalPort != tc.wantLocalPort {
					t.Errorf("LocalPort = %d, want %d", cfg.LocalPort, tc.wantLocalPort)
				}
				if cfg.RemoteHost != tc.wantRemoteHost {
					t.Errorf("RemoteHost = %q, want %q", cfg.RemoteHost, tc.wantRemoteHost)
				}
				if cfg.RemotePort != tc.wantRemotePort {
					t.Errorf("RemotePort = %d, want %d", cfg.RemotePort, tc.wantRemotePort)
				}
			}
		})
	}
}

func TestNewRemotePortForwardingCfg(t *testing.T) {
	t.Parallel()

	// Remote config reverses local and remote
	cfg := newRemotePortForwardingCfg("localhost:8080:remote.host:80")

	if cfg.LocalHost != "remote.host" {
		t.Errorf("LocalHost = %q, want %q", cfg.LocalHost, "remote.host")
	}
	if cfg.LocalPort != 80 {
		t.Errorf("LocalPort = %d, want %d", cfg.LocalPort, 80)
	}
	if cfg.RemoteHost != "localhost" {
		t.Errorf("RemoteHost = %q, want %q", cfg.RemoteHost, "localhost")
	}
	if cfg.RemotePort != 8080 {
		t.Errorf("RemotePort = %d, want %d", cfg.RemotePort, 8080)
	}
}

func TestLocalPortForwardingCfg_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *LocalPortForwardingCfg
		want string
	}{
		{
			name: "valid config",
			cfg: &LocalPortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "localhost:8080:remote.host:80",
		},
		{
			name: "config with parsing error",
			cfg: &LocalPortForwardingCfg{
				spec:       "invalid:spec",
				parsingErr: ErrInvalidFormat,
			},
			want: "invalid:spec",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.cfg.String(); got != tc.want {
				t.Errorf("LocalPortForwardingCfg.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRemotePortForwardingCfg_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *RemotePortForwardingCfg
		want string
	}{
		{
			name: "valid config",
			cfg: &RemotePortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "remote.host:80:localhost:8080",
		},
		{
			name: "config with parsing error",
			cfg: &RemotePortForwardingCfg{
				spec:       "invalid:spec",
				parsingErr: ErrInvalidFormat,
			},
			want: "invalid:spec",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.cfg.String(); got != tc.want {
				t.Errorf("RemotePortForwardingCfg.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLocalPortForwardingCfg_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *LocalPortForwardingCfg
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &LocalPortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			wantErr: false,
		},
		{
			name: "empty remote host",
			cfg: &LocalPortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "",
				RemotePort: 80,
			},
			wantErr: true,
		},
		{
			name: "invalid local port",
			cfg: &LocalPortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  0,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			wantErr: true,
		},
		{
			name: "invalid remote port",
			cfg: &LocalPortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 65536,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := tc.cfg.validate()
			if (len(errs) > 0) != tc.wantErr {
				t.Errorf("LocalPortForwardingCfg.validate() errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}

func TestRemotePortForwardingCfg_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *RemotePortForwardingCfg
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &RemotePortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			wantErr: false,
		},
		{
			name: "empty local host",
			cfg: &RemotePortForwardingCfg{
				LocalHost:  "",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			wantErr: true,
		},
		{
			name: "invalid ports",
			cfg: &RemotePortForwardingCfg{
				LocalHost:  "localhost",
				LocalPort:  0,
				RemoteHost: "remote.host",
				RemotePort: 65536,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := tc.cfg.validate()
			if (len(errs) > 0) != tc.wantErr {
				t.Errorf("RemotePortForwardingCfg.validate() errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}
