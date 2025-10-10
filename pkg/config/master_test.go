package config

import (
	"testing"
)

func TestMaster_ParseLocalPortForwardingSpecs(t *testing.T) {
	t.Parallel()

	m := &Master{}
	specs := []string{"8080:remote:80", "localhost:9090:example.com:443"}
	m.ParseLocalPortForwardingSpecs(specs)

	if len(m.LocalPortForwarding) != 2 {
		t.Errorf("ParseLocalPortForwardingSpecs() added %d configs, want 2", len(m.LocalPortForwarding))
	}
}

func TestMaster_ParseRemotePortForwardingSpecs(t *testing.T) {
	t.Parallel()

	m := &Master{}
	specs := []string{"8080:remote:80", "localhost:9090:example.com:443"}
	m.ParseRemotePortForwardingSpecs(specs)

	if len(m.RemotePortForwarding) != 2 {
		t.Errorf("ParseRemotePortForwardingSpecs() added %d configs, want 2", len(m.RemotePortForwarding))
	}

	// Check that destinations are tracked
	if !m.IsAllowedRemotePortForwardingDestination("remote", 80) {
		t.Error("Expected remote:80 to be an allowed destination")
	}
	if !m.IsAllowedRemotePortForwardingDestination("example.com", 443) {
		t.Error("Expected example.com:443 to be an allowed destination")
	}
	if m.IsAllowedRemotePortForwardingDestination("notallowed", 1234) {
		t.Error("Expected notallowed:1234 to not be an allowed destination")
	}
}

func TestMaster_IsAllowedRemotePortForwardingDestination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		specs   []string
		host    string
		port    int
		allowed bool
	}{
		{
			name:    "empty specs - not allowed",
			specs:   []string{},
			host:    "example.com",
			port:    80,
			allowed: false,
		},
		{
			name:    "matching destination",
			specs:   []string{"8080:example.com:80"},
			host:    "example.com",
			port:    80,
			allowed: true,
		},
		{
			name:    "non-matching host",
			specs:   []string{"8080:example.com:80"},
			host:    "other.com",
			port:    80,
			allowed: false,
		},
		{
			name:    "non-matching port",
			specs:   []string{"8080:example.com:80"},
			host:    "example.com",
			port:    443,
			allowed: false,
		},
		{
			name:    "multiple specs - match second",
			specs:   []string{"8080:first.com:80", "9090:second.com:443"},
			host:    "second.com",
			port:    443,
			allowed: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &Master{}
			m.ParseRemotePortForwardingSpecs(tc.specs)

			if got := m.IsAllowedRemotePortForwardingDestination(tc.host, tc.port); got != tc.allowed {
				t.Errorf("IsAllowedRemotePortForwardingDestination(%q, %d) = %v, want %v", tc.host, tc.port, got, tc.allowed)
			}
		})
	}
}

func TestMaster_IsSocksEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		socks   *SocksCfg
		enabled bool
	}{
		{
			name:    "socks nil - disabled",
			socks:   nil,
			enabled: false,
		},
		{
			name:    "socks configured - enabled",
			socks:   &SocksCfg{Host: "localhost", Port: 1080},
			enabled: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &Master{Socks: tc.socks}
			if got := m.IsSocksEnabled(); got != tc.enabled {
				t.Errorf("IsSocksEnabled() = %v, want %v", got, tc.enabled)
			}
		})
	}
}

func TestMaster_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		master  *Master
		wantErr bool
	}{
		{
			name:    "empty config - valid",
			master:  &Master{},
			wantErr: false,
		},
		{
			name: "valid local port forwarding",
			master: &Master{
				LocalPortForwarding: []*LocalPortForwardingCfg{
					{
						LocalPort:  8080,
						RemoteHost: "example.com",
						RemotePort: 80,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid local port forwarding - bad port",
			master: &Master{
				LocalPortForwarding: []*LocalPortForwardingCfg{
					{
						LocalPort:  0,
						RemoteHost: "example.com",
						RemotePort: 80,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid local port forwarding - parsing error",
			master: &Master{
				LocalPortForwarding: []*LocalPortForwardingCfg{
					{
						spec:       "invalid",
						parsingErr: ErrInvalidFormat,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid remote port forwarding",
			master: &Master{
				RemotePortForwarding: []*RemotePortForwardingCfg{
					{
						LocalHost:  "localhost",
						LocalPort:  8080,
						RemotePort: 80,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid remote port forwarding - empty local host",
			master: &Master{
				RemotePortForwarding: []*RemotePortForwardingCfg{
					{
						LocalHost:  "",
						LocalPort:  8080,
						RemotePort: 80,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid socks config",
			master: &Master{
				Socks: &SocksCfg{
					Host: "localhost",
					Port: 1080,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid socks config - parsing error",
			master: &Master{
				Socks: &SocksCfg{
					spec:       "invalid:spec:format",
					parsingErr: ErrInvalidFormat,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid socks config - bad port",
			master: &Master{
				Socks: &SocksCfg{
					Host: "localhost",
					Port: 0,
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := tc.master.Validate()
			if (len(errs) > 0) != tc.wantErr {
				t.Errorf("Master.Validate() errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}
