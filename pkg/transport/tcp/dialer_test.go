package tcp

import (
	"net"
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
			d, err := NewDialer(tc.addr)
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

func TestDialer_Dial(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	// Start a local listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Accept one connection
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	// Test dialing
	d, err := NewDialer(addr)
	if err != nil {
		t.Fatalf("NewDialer() error = %v", err)
	}

	conn, err := d.Dial()
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	if conn == nil {
		t.Fatal("Dial() returned nil connection")
	}
	defer conn.Close()

	// Verify it's a TCP connection
	if _, ok := conn.(*net.TCPConn); !ok {
		t.Error("Dial() did not return a TCPConn")
	}
}

func TestDialer_Dial_Failure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	// Try to dial a non-existent server
	d, err := NewDialer("127.0.0.1:1")
	if err != nil {
		t.Fatalf("NewDialer() error = %v", err)
	}

	_, err = d.Dial()
	if err == nil {
		t.Error("Dial() expected error for non-existent server, got nil")
	}
}
