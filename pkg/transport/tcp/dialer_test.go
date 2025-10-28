package tcp

import (
	"testing"
)

func TestDial_AddressParsing(t *testing.T) {
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
			// Test address parsing by calling resolveTCPAddress directly
			_, err := resolveTCPAddress(tc.addr)
			if (err != nil) != tc.wantErr {
				t.Errorf("resolveTCPAddress(%q) error = %v, wantErr %v", tc.addr, err, tc.wantErr)
			}
		})
	}
}
