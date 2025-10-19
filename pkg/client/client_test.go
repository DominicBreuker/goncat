package client

import (
	"context"
	"crypto/x509"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/transport"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

// Test helpers and fakes

// fakeDialer implements transport.Dialer for testing.
type fakeDialer struct {
	dialErr error
	conn    net.Conn
}

func (f *fakeDialer) Dial(ctx context.Context) (net.Conn, error) {
	if f.dialErr != nil {
		return nil, f.dialErr
	}
	return f.conn, nil
}

// fakeConn implements net.Conn for testing.
type fakeConn struct {
	closed  bool
	closeCh chan struct{}
	mu      sync.Mutex
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *fakeConn) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		if f.closeCh != nil {
			close(f.closeCh)
		}
	}
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (f *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}
}

func (f *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (f *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (f *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// Test basic client creation

func TestNew(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	client := New(ctx, cfg)

	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.ctx != ctx {
		t.Error("New() did not set context correctly")
	}
	if client.cfg != cfg {
		t.Error("New() did not set config correctly")
	}
}

func TestClient_GetConnection_Nil(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	client := New(ctx, cfg)
	conn := client.GetConnection()

	if conn != nil {
		t.Error("GetConnection() expected nil before Connect(), got non-nil")
	}
}

// Test TCP dialer creation and connection

func TestConnect_TCPDialerCreation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Timeout:  time.Second,
	}

	client := New(ctx, cfg)

	fakeConn := &fakeConn{}
	fakeDialer := &fakeDialer{conn: fakeConn}

	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			if addr != "localhost:8080" {
				t.Errorf("newTCPDialer got addr %s, want localhost:8080", addr)
			}
			return fakeDialer, nil
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			t.Error("newWSDialer should not be called for TCP protocol")
			return nil
		},
		tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
			t.Error("tlsUpgrader should not be called when SSL is false")
			return nil, nil
		},
	}

	err := client.connect(deps)
	if err != nil {
		t.Fatalf("connect() error = %v, want nil", err)
	}

	if client.conn != fakeConn {
		t.Error("connect() did not set connection correctly")
	}
}

func TestConnect_TCPDialerCreationError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Timeout:  time.Second,
	}

	client := New(ctx, cfg)

	expectedErr := errors.New("dialer creation failed")

	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			return nil, expectedErr
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			t.Error("newWSDialer should not be called for TCP protocol")
			return nil
		},
		tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
			t.Error("tlsUpgrader should not be called when dialer creation fails")
			return nil, nil
		},
	}

	err := client.connect(deps)
	if err == nil {
		t.Fatal("connect() error = nil, want error")
	}
	if !errors.Is(err, expectedErr) && err.Error() != "NewDialer: dialer creation failed" {
		t.Errorf("connect() error = %v, want wrapped dialer creation error", err)
	}
}

func TestConnect_TCPDialError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Timeout:  time.Second,
	}

	client := New(ctx, cfg)

	dialErr := errors.New("dial failed")
	fakeDialer := &fakeDialer{dialErr: dialErr}

	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			return fakeDialer, nil
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			t.Error("newWSDialer should not be called for TCP protocol")
			return nil
		},
		tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
			t.Error("tlsUpgrader should not be called when dial fails")
			return nil, nil
		},
	}

	err := client.connect(deps)
	if err == nil {
		t.Fatal("connect() error = nil, want error")
	}
	if !errors.Is(err, dialErr) && err.Error() != "Dial(): dial failed" {
		t.Errorf("connect() error = %v, want wrapped dial error", err)
	}
}

// Test WebSocket dialer creation and connection

func TestConnect_WSDialerCreation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		protocol config.Protocol
	}{
		{"WebSocket", config.ProtoWS},
		{"WebSocket Secure", config.ProtoWSS},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			cfg := &config.Shared{
				Protocol: tc.protocol,
				Host:     "example.com",
				Port:     443,
				Timeout:  time.Second,
			}

			client := New(ctx, cfg)

			fakeConn := &fakeConn{}
			fakeDialer := &fakeDialer{conn: fakeConn}

			deps := &dependencies{
				newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
					t.Error("newTCPDialer should not be called for WebSocket protocol")
					return nil, nil
				},
				newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
					if addr != "example.com:443" {
						t.Errorf("newWSDialer got addr %s, want example.com:443", addr)
					}
					if proto != tc.protocol {
						t.Errorf("newWSDialer got proto %v, want %v", proto, tc.protocol)
					}
					return fakeDialer
				},
				tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
					t.Error("tlsUpgrader should not be called when SSL is false")
					return nil, nil
				},
			}

			err := client.connect(deps)
			if err != nil {
				t.Fatalf("connect() error = %v, want nil", err)
			}

			if client.conn != fakeConn {
				t.Error("connect() did not set connection correctly")
			}
		})
	}
}

func TestConnect_WSDialError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoWS,
		Host:     "example.com",
		Port:     8080,
		Timeout:  time.Second,
	}

	client := New(ctx, cfg)

	dialErr := errors.New("websocket dial failed")
	fakeDialer := &fakeDialer{dialErr: dialErr}

	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			t.Error("newTCPDialer should not be called for WebSocket protocol")
			return nil, nil
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			return fakeDialer
		},
		tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
			t.Error("tlsUpgrader should not be called when dial fails")
			return nil, nil
		},
	}

	err := client.connect(deps)
	if err == nil {
		t.Fatal("connect() error = nil, want error")
	}
	if !errors.Is(err, dialErr) && err.Error() != "Dial(): websocket dial failed" {
		t.Errorf("connect() error = %v, want wrapped dial error", err)
	}
}

// Test TLS upgrade

func TestConnect_TLSUpgrade(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Timeout:  time.Second,
	}

	client := New(ctx, cfg)

	plainConn := &fakeConn{}
	tlsConn := &fakeConn{}
	fakeDialer := &fakeDialer{conn: plainConn}

	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			return fakeDialer, nil
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			t.Error("newWSDialer should not be called for TCP protocol")
			return nil
		},
		tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
			if conn != plainConn {
				t.Error("tlsUpgrader got wrong connection")
			}
			if key != "" {
				t.Error("tlsUpgrader expected empty key when no key is configured")
			}
			if timeout != time.Second {
				t.Errorf("tlsUpgrader got timeout %v, want %v", timeout, time.Second)
			}
			return tlsConn, nil
		},
	}

	err := client.connect(deps)
	if err != nil {
		t.Fatalf("connect() error = %v, want nil", err)
	}

	if client.conn != tlsConn {
		t.Error("connect() did not set TLS connection correctly")
	}
}

func TestConnect_TLSUpgradeError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Timeout:  time.Second,
	}

	client := New(ctx, cfg)

	fakeConn := &fakeConn{}
	fakeDialer := &fakeDialer{conn: fakeConn}
	tlsErr := errors.New("tls upgrade failed")

	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			return fakeDialer, nil
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			t.Error("newWSDialer should not be called for TCP protocol")
			return nil
		},
		tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
			return nil, tlsErr
		},
	}

	err := client.connect(deps)
	if err == nil {
		t.Fatal("connect() error = nil, want error")
	}
	if !errors.Is(err, tlsErr) && err.Error() != "upgradeToTLS: tls upgrade failed" {
		t.Errorf("connect() error = %v, want wrapped TLS error", err)
	}
}

// Test TLS upgrade with mutual authentication

func TestConnect_TLSUpgradeWithMTLS(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Key:      "test-key",
		Timeout:  time.Second,
	}

	client := New(ctx, cfg)

	plainConn := &fakeConn{}
	tlsConn := &fakeConn{}
	fakeDialer := &fakeDialer{conn: plainConn}

	deps := &dependencies{
		newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
			return fakeDialer, nil
		},
		newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
			t.Error("newWSDialer should not be called for TCP protocol")
			return nil
		},
		tlsUpgrader: func(conn net.Conn, key string, timeout time.Duration) (net.Conn, error) {
			if conn != plainConn {
				t.Error("tlsUpgrader got wrong connection")
			}
			// Key is salted by cfg.GetKey()
			if key == "" {
				t.Error("tlsUpgrader expected non-empty key when key is configured")
			}
			if timeout != time.Second {
				t.Errorf("tlsUpgrader got timeout %v, want %v", timeout, time.Second)
			}
			return tlsConn, nil
		},
	}

	err := client.connect(deps)
	if err != nil {
		t.Fatalf("connect() error = %v, want nil", err)
	}

	if client.conn != tlsConn {
		t.Error("connect() did not set TLS connection correctly")
	}
}

// Test customVerifier

func TestCustomVerifier_InvalidCertCount(t *testing.T) {
	t.Parallel()

	caCert := x509.NewCertPool()

	tests := []struct {
		name     string
		rawCerts [][]byte
		wantErr  bool
	}{
		{
			name:     "no certificates",
			rawCerts: [][]byte{},
			wantErr:  true,
		},
		{
			name:     "multiple certificates",
			rawCerts: [][]byte{{}, {}},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := customVerifier(caCert, tc.rawCerts)
			if (err != nil) != tc.wantErr {
				t.Errorf("customVerifier() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestCustomVerifier_InvalidCertificate(t *testing.T) {
	t.Parallel()

	caCert := x509.NewCertPool()

	// Invalid certificate data
	rawCerts := [][]byte{{0x00, 0x01, 0x02}}

	err := customVerifier(caCert, rawCerts)
	if err == nil {
		t.Error("customVerifier() expected error for invalid certificate data, got nil")
	}
}

// Test upgradeToTLS with timeout enforcement

func TestUpgradeToTLS_HandshakeTimeout(t *testing.T) {
	t.Parallel()

	// Create a fake connection that tracks SetDeadline calls
	conn := &fakeConnWithDeadline{
		handshakeErr: errors.New("handshake timeout"),
	}

	_, err := upgradeToTLS(conn, "", 1*time.Second)
	if err == nil {
		t.Fatal("upgradeToTLS() error = nil, want error")
	}

	if !conn.deadlineSet {
		t.Error("upgradeToTLS() did not set deadline before handshake")
	}
}

// fakeConnWithDeadline tracks SetDeadline calls for testing.
type fakeConnWithDeadline struct {
	fakeConn
	deadlineSet  bool
	handshakeErr error
}

func (f *fakeConnWithDeadline) SetDeadline(t time.Time) error {
	if !t.IsZero() {
		f.deadlineSet = true
	}
	return nil
}

// Test TLS config options

func TestUpgradeToTLS_TLSConfig(t *testing.T) {
	t.Parallel()

	// We can't easily test the full TLS handshake without a real server,
	// but we can verify that the function is called with the right parameters
	// by using the dependency injection pattern in connect().
	// This test is covered by the integration tests with real TLS connections.

	// Here we just verify the function signature and basic error handling
	conn := &fakeConnWithDeadline{}

	// Test without key (should create basic TLS config)
	_, err := upgradeToTLS(conn, "", 1*time.Second)
	// We expect an error because fakeConn doesn't implement a real TLS handshake,
	// but we're testing that the function doesn't panic and attempts the upgrade
	if err == nil {
		t.Error("upgradeToTLS() with fake conn should fail handshake")
	}
}

func TestUpgradeToTLS_WithKey(t *testing.T) {
	t.Parallel()

	conn := &fakeConnWithDeadline{}

	// Test with key (should create mTLS config)
	_, err := upgradeToTLS(conn, "test-key", 1*time.Second)
	// We expect an error because:
	// 1. fakeConn doesn't implement a real TLS handshake, OR
	// 2. The certificate generation might succeed but handshake will fail
	if err == nil {
		t.Error("upgradeToTLS() with fake conn should fail handshake")
	}
}
