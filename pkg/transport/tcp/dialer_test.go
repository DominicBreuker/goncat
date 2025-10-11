package tcp

import (
	"dominicbreuker/goncat/mocks"
	"dominicbreuker/goncat/pkg/config"
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

func TestDialer_Dial(t *testing.T) {
	// Use mock TCP network instead of real network
	mockNet := mocks.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
	}

	addr := "127.0.0.1:12345"

	// Create a listener on the mock network
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	listener, err := mockNet.ListenTCP("tcp", tcpAddr)
	if err != nil {
		t.Fatalf("Failed to create mock listener: %v", err)
	}
	defer listener.Close()

	// Accept one connection
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	// Test dialing with mock network
	d, err := NewDialer(addr, deps)
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

	// Verify connection works by writing and reading
	testData := []byte("hello")
	go func() {
		serverConn, _ := listener.Accept()
		if serverConn != nil {
			buf := make([]byte, len(testData))
			serverConn.Read(buf)
			serverConn.Close()
		}
	}()

	_, err = conn.Write(testData)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
}

func TestDialer_Dial_Failure(t *testing.T) {
	// Use mock TCP network instead of real network
	mockNet := mocks.NewMockTCPNetwork()
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
	}

	// Try to dial a non-existent server (no listener created)
	d, err := NewDialer("127.0.0.1:1", deps)
	if err != nil {
		t.Fatalf("NewDialer() error = %v", err)
	}

	_, err = d.Dial()
	if err == nil {
		t.Error("Dial() expected error for non-existent server, got nil")
	}
}
