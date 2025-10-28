package ws

import (
	"testing"
)

func TestFormatURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		protocol string
		addr     string
		wantURL  string
	}{
		{
			name:     "ws protocol",
			protocol: "ws",
			addr:     "localhost:8080",
			wantURL:  "ws://localhost:8080",
		},
		{
			name:     "wss protocol",
			protocol: "wss",
			addr:     "example.com:443",
			wantURL:  "wss://example.com:443",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			url := formatURL(tc.protocol, tc.addr)
			if url != tc.wantURL {
				t.Errorf("formatURL(%q, %q) = %q, want %q", tc.protocol, tc.addr, url, tc.wantURL)
			}
		})
	}
}
