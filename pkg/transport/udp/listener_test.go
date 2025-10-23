package udp

import (
	"testing"
)

func TestNewListener(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "valid address with port 0",
			addr:    "127.0.0.1:0",
			wantErr: false,
		},
		{
			name:    "wildcard address",
			addr:    ":0",
			wantErr: false,
		},
		{
			name:    "invalid address",
			addr:    "not-a-valid-address",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l, err := NewListener(tc.addr, nil)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewListener(%q) error = %v, wantErr %v", tc.addr, err, tc.wantErr)
			}
			if !tc.wantErr && l == nil {
				t.Error("NewListener() returned nil listener")
			}
			if !tc.wantErr && l != nil {
				// Clean up
				_ = l.Close()
			}
		})
	}
}
