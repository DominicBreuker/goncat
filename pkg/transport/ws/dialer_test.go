package ws

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"testing"
)

func TestNewDialer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		proto   config.Protocol
		wantURL string
	}{
		{
			name:    "ws protocol",
			addr:    "localhost:8080",
			proto:   config.ProtoWS,
			wantURL: "ws://localhost:8080",
		},
		{
			name:    "wss protocol",
			addr:    "example.com:443",
			proto:   config.ProtoWSS,
			wantURL: "wss://example.com:443",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			d := NewDialer(ctx, tc.addr, tc.proto)
			if d == nil {
				t.Fatal("NewDialer() returned nil")
			}
			if d.url != tc.wantURL {
				t.Errorf("NewDialer() url = %q, want %q", d.url, tc.wantURL)
			}
		})
	}
}
