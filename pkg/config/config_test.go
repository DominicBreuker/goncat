package config

import (
	"testing"
)

func TestProtocol_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		protocol Protocol
		want     string
	}{
		{"TCP", ProtoTCP, "tcp"},
		{"WebSocket", ProtoWS, "ws"},
		{"WebSocket Secure", ProtoWSS, "wss"},
		{"Invalid", Protocol(999), ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.protocol.String(); got != tc.want {
				t.Errorf("Protocol.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestShared_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *Shared
		wantErr bool
	}{
		{
			name: "valid config with SSL and key",
			cfg: &Shared{
				Protocol: ProtoTCP,
				Host:     "localhost",
				Port:     8080,
				SSL:      true,
				Key:      "secret",
			},
			wantErr: false,
		},
		{
			name: "valid config without SSL",
			cfg: &Shared{
				Protocol: ProtoTCP,
				Host:     "localhost",
				Port:     8080,
				SSL:      false,
				Key:      "",
			},
			wantErr: false,
		},
		{
			name: "invalid: key without SSL",
			cfg: &Shared{
				Protocol: ProtoTCP,
				Host:     "localhost",
				Port:     8080,
				SSL:      false,
				Key:      "secret",
			},
			wantErr: true,
		},
		{
			name: "invalid: port too low",
			cfg: &Shared{
				Protocol: ProtoTCP,
				Host:     "localhost",
				Port:     0,
				SSL:      false,
			},
			wantErr: true,
		},
		{
			name: "invalid: port too high",
			cfg: &Shared{
				Protocol: ProtoTCP,
				Host:     "localhost",
				Port:     65536,
				SSL:      false,
			},
			wantErr: true,
		},
		{
			name: "valid: port 1",
			cfg: &Shared{
				Protocol: ProtoTCP,
				Host:     "localhost",
				Port:     1,
				SSL:      false,
			},
			wantErr: false,
		},
		{
			name: "valid: port 65535",
			cfg: &Shared{
				Protocol: ProtoTCP,
				Host:     "localhost",
				Port:     65535,
				SSL:      false,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := tc.cfg.Validate()
			if (len(errs) > 0) != tc.wantErr {
				t.Errorf("Shared.Validate() errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}

func TestShared_GetKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *Shared
		want string
	}{
		{
			name: "empty key",
			cfg:  &Shared{Key: ""},
			want: "",
		},
		{
			name: "with key",
			cfg:  &Shared{Key: "mykey"},
			want: KeySalt + "mykey",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.cfg.GetKey(); got != tc.want {
				t.Errorf("Shared.GetKey() = %q, want %q", got, tc.want)
			}
		})
	}
}
