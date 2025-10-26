package config

import (
	"testing"
)

func TestNewLocalPortForwardingCfg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           string
		wantProtocol   string
		wantLocalHost  string
		wantLocalPort  int
		wantRemoteHost string
		wantRemotePort int
		wantError      bool
	}{
		{
			name:           "three parts",
			spec:           "8080:remote.host:80",
			wantProtocol:   "tcp",
			wantLocalHost:  "",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "four parts",
			spec:           "localhost:8080:remote.host:80",
			wantProtocol:   "tcp",
			wantLocalHost:  "localhost",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "TCP prefix uppercase",
			spec:           "T:8080:remote.host:80",
			wantProtocol:   "tcp",
			wantLocalHost:  "",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "TCP prefix lowercase",
			spec:           "t:8080:remote.host:80",
			wantProtocol:   "tcp",
			wantLocalHost:  "",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "UDP prefix uppercase",
			spec:           "U:8080:remote.host:80",
			wantProtocol:   "udp",
			wantLocalHost:  "",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "UDP prefix lowercase",
			spec:           "u:8080:remote.host:80",
			wantProtocol:   "udp",
			wantLocalHost:  "",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "TCP prefix with local host",
			spec:           "T:localhost:8080:remote.host:80",
			wantProtocol:   "tcp",
			wantLocalHost:  "localhost",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "UDP prefix with local host",
			spec:           "U:localhost:8080:remote.host:80",
			wantProtocol:   "udp",
			wantLocalHost:  "localhost",
			wantLocalPort:  8080,
			wantRemoteHost: "remote.host",
			wantRemotePort: 80,
			wantError:      false,
		},
		{
			name:           "invalid format - too few parts",
			spec:           "8080:80",
			wantProtocol:   "",
			wantLocalHost:  "",
			wantLocalPort:  0,
			wantRemoteHost: "",
			wantRemotePort: 0,
			wantError:      true,
		},
		{
			name:           "invalid format - too many parts",
			spec:           "a:b:c:d:e:f",
			wantProtocol:   "",
			wantLocalHost:  "",
			wantLocalPort:  0,
			wantRemoteHost: "",
			wantRemotePort: 0,
			wantError:      true,
		},
		{
			name:           "invalid local port",
			spec:           "abc:remote.host:80",
			wantProtocol:   "",
			wantLocalHost:  "",
			wantLocalPort:  0,
			wantRemoteHost: "",
			wantRemotePort: 0,
			wantError:      true,
		},
		{
			name:           "invalid remote port",
			spec:           "8080:remote.host:abc",
			wantProtocol:   "",
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
				if cfg.Protocol != tc.wantProtocol {
					t.Errorf("Protocol = %q, want %q", cfg.Protocol, tc.wantProtocol)
				}
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

	tests := []struct {
		name           string
		spec           string
		wantProtocol   string
		wantLocalHost  string
		wantLocalPort  int
		wantRemoteHost string
		wantRemotePort int
	}{
		{
			name:           "TCP implicit",
			spec:           "localhost:8080:remote.host:80",
			wantProtocol:   "tcp",
			wantLocalHost:  "remote.host",
			wantLocalPort:  80,
			wantRemoteHost: "localhost",
			wantRemotePort: 8080,
		},
		{
			name:           "TCP explicit",
			spec:           "T:localhost:8080:remote.host:80",
			wantProtocol:   "tcp",
			wantLocalHost:  "remote.host",
			wantLocalPort:  80,
			wantRemoteHost: "localhost",
			wantRemotePort: 8080,
		},
		{
			name:           "UDP explicit",
			spec:           "U:localhost:8080:remote.host:80",
			wantProtocol:   "udp",
			wantLocalHost:  "remote.host",
			wantLocalPort:  80,
			wantRemoteHost: "localhost",
			wantRemotePort: 8080,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := newRemotePortForwardingCfg(tc.spec)

			if cfg.Protocol != tc.wantProtocol {
				t.Errorf("Protocol = %q, want %q", cfg.Protocol, tc.wantProtocol)
			}
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
		})
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
			name: "valid config TCP",
			cfg: &LocalPortForwardingCfg{
				Protocol:   "tcp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "localhost:8080:remote.host:80",
		},
		{
			name: "valid config UDP",
			cfg: &LocalPortForwardingCfg{
				Protocol:   "udp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "U:localhost:8080:remote.host:80",
		},
		{
			name: "valid config TCP no local host",
			cfg: &LocalPortForwardingCfg{
				Protocol:   "tcp",
				LocalHost:  "",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "8080:remote.host:80",
		},
		{
			name: "valid config UDP no local host",
			cfg: &LocalPortForwardingCfg{
				Protocol:   "udp",
				LocalHost:  "",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "U:8080:remote.host:80",
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
			name: "valid config TCP",
			cfg: &RemotePortForwardingCfg{
				Protocol:   "tcp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "remote.host:80:localhost:8080",
		},
		{
			name: "valid config UDP",
			cfg: &RemotePortForwardingCfg{
				Protocol:   "udp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			want: "U:remote.host:80:localhost:8080",
		},
		{
			name: "valid config TCP no remote host",
			cfg: &RemotePortForwardingCfg{
				Protocol:   "tcp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "",
				RemotePort: 80,
			},
			want: "80:localhost:8080",
		},
		{
			name: "valid config UDP no remote host",
			cfg: &RemotePortForwardingCfg{
				Protocol:   "udp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "",
				RemotePort: 80,
			},
			want: "U:80:localhost:8080",
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
			name: "valid config TCP",
			cfg: &LocalPortForwardingCfg{
				Protocol:   "tcp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			wantErr: false,
		},
		{
			name: "valid config UDP",
			cfg: &LocalPortForwardingCfg{
				Protocol:   "udp",
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
				Protocol:   "tcp",
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
				Protocol:   "tcp",
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
				Protocol:   "tcp",
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
			name: "valid config TCP",
			cfg: &RemotePortForwardingCfg{
				Protocol:   "tcp",
				LocalHost:  "localhost",
				LocalPort:  8080,
				RemoteHost: "remote.host",
				RemotePort: 80,
			},
			wantErr: false,
		},
		{
			name: "valid config UDP",
			cfg: &RemotePortForwardingCfg{
				Protocol:   "udp",
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
				Protocol:   "tcp",
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
				Protocol:   "tcp",
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
