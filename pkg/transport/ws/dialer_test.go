package ws

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
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

func TestDialer_Dial(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	// Create a test WebSocket server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{"bin"},
		})
		if err != nil {
			t.Logf("websocket.Accept() error: %v", err)
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		// Echo back any messages
		ctx := context.Background()
		for {
			_, msg, err := c.Read(ctx)
			if err != nil {
				return
			}
			if err := c.Write(ctx, websocket.MessageBinary, msg); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// Extract address from server URL (format: http://127.0.0.1:port)
	addr := strings.TrimPrefix(server.URL, "http://")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d := NewDialer(ctx, addr, config.ProtoWS)
	conn, err := d.Dial()
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	if conn == nil {
		t.Fatal("Dial() returned nil connection")
	}
	defer conn.Close()

	// Test that the connection works
	testMsg := []byte("hello")
	_, err = conn.Write(testMsg)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	buf := make([]byte, len(testMsg))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if string(buf[:n]) != string(testMsg) {
		t.Errorf("Read() = %q, want %q", buf[:n], testMsg)
	}
}

func TestDialer_Dial_Failure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to dial a non-existent server
	d := NewDialer(ctx, "127.0.0.1:1", config.ProtoWS)
	_, err := d.Dial()
	if err == nil {
		t.Error("Dial() expected error for non-existent server, got nil")
	}
}
