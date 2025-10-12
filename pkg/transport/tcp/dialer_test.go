package tcp

import (
	"testing"
)

func TestNewDialer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "valid address",
			addr:    "localhost:8080",
			wantErr: false,
		},
		{
			name:    "valid IPv4 address",
			addr:    "127.0.0.1:8080",
			wantErr: false,
		},
		{
			name:    "valid IPv6 address",
			addr:    "[::1]:8080",
			wantErr: false,
		},
		{
			name:    "invalid address - no port",
			addr:    "localhost",
			wantErr: true,
		},
		{
			name:    "invalid address - bad port",
			addr:    "localhost:abc",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d, err := NewDialer(tc.addr, nil)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewDialer(%q) error = %v, wantErr %v", tc.addr, err, tc.wantErr)
			}
			if !tc.wantErr && d == nil {
				t.Error("NewDialer() returned nil dialer")
			}
			if !tc.wantErr && d.tcpAddr == nil {
				t.Error("NewDialer() dialer has nil tcpAddr")
			}
		})
	}
}
