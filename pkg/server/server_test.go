package server

import (
	"context"
	"crypto/tls"
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

// fakeListener implements transport.Listener for testing.
type fakeListener struct {
	serveErr error
	closed   bool
	mu       sync.Mutex
}

func (f *fakeListener) Serve(handle transport.Handler) error {
	return f.serveErr
}

func (f *fakeListener) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// Test basic server creation

func TestNew(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      false,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	s, err := New(ctx, cfg, handler)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if s == nil {
		t.Fatal("New() returned nil server")
	}
	if s.ctx != ctx {
		t.Error("New() did not set context correctly")
	}
	if s.cfg != cfg {
		t.Error("New() did not set config correctly")
	}
}

// Test TCP listener creation

func TestServe_TCPListenerCreation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	fakeListener := &fakeListener{}

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			if addr != "localhost:8080" {
				t.Errorf("newTCPListener got addr %s, want localhost:8080", addr)
			}
			return fakeListener, nil
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			t.Error("newWSListener should not be called for TCP protocol")
			return nil, nil
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			t.Error("certGenerator should not be called when SSL is false")
			return nil, tls.Certificate{}, nil
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("newServer() error = %v, want nil", err)
	}

	err = s.Serve()
	if err != nil {
		t.Errorf("Serve() error = %v, want nil", err)
	}

	if s.l != fakeListener {
		t.Error("Serve() did not set listener correctly")
	}
}

func TestServe_TCPListenerCreationError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	expectedErr := errors.New("listener creation failed")

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			return nil, expectedErr
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			t.Error("newWSListener should not be called for TCP protocol")
			return nil, nil
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			t.Error("certGenerator should not be called when SSL is false")
			return nil, tls.Certificate{}, nil
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("newServer() error = %v, want nil", err)
	}

	err = s.Serve()
	if err == nil {
		t.Fatal("Serve() error = nil, want error")
	}
	if !errors.Is(err, expectedErr) && err.Error() != "tcp.New(localhost:8080): listener creation failed" {
		t.Errorf("Serve() error = %v, want wrapped listener creation error", err)
	}
}

func TestServe_TCPServeError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	serveErr := errors.New("serve failed")
	fakeListener := &fakeListener{serveErr: serveErr}

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			return fakeListener, nil
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			t.Error("newWSListener should not be called for TCP protocol")
			return nil, nil
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			t.Error("certGenerator should not be called when SSL is false")
			return nil, tls.Certificate{}, nil
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("newServer() error = %v, want nil", err)
	}

	err = s.Serve()
	if err == nil {
		t.Fatal("Serve() error = nil, want error")
	}
	if !errors.Is(err, serveErr) {
		t.Errorf("Serve() error = %v, want serve error", err)
	}
}

// Test WebSocket listener creation

func TestServe_WSListenerCreation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		protocol config.Protocol
		secure   bool
	}{
		{"WebSocket", config.ProtoWS, false},
		{"WebSocket Secure", config.ProtoWSS, true},
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

			handler := func(conn net.Conn) error {
				return nil
			}

			fakeListener := &fakeListener{}

			deps := &dependencies{
				newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
					t.Error("newTCPListener should not be called for WebSocket protocol")
					return nil, nil
				},
				newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
					if addr != "example.com:443" {
						t.Errorf("newWSListener got addr %s, want example.com:443", addr)
					}
					if secure != tc.secure {
						t.Errorf("newWSListener got secure %v, want %v", secure, tc.secure)
					}
					return fakeListener, nil
				},
				certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
					t.Error("certGenerator should not be called when SSL is false")
					return nil, tls.Certificate{}, nil
				},
			}

			s, err := newServer(ctx, cfg, handler, deps)
			if err != nil {
				t.Fatalf("newServer() error = %v, want nil", err)
			}

			err = s.Serve()
			if err != nil {
				t.Errorf("Serve() error = %v, want nil", err)
			}

			if s.l != fakeListener {
				t.Error("Serve() did not set listener correctly")
			}
		})
	}
}

func TestServe_WSListenerCreationError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoWS,
		Host:     "example.com",
		Port:     8080,
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	expectedErr := errors.New("websocket listener creation failed")

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			t.Error("newTCPListener should not be called for WebSocket protocol")
			return nil, nil
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			return nil, expectedErr
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			t.Error("certGenerator should not be called when SSL is false")
			return nil, tls.Certificate{}, nil
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("newServer() error = %v, want nil", err)
	}

	err = s.Serve()
	if err == nil {
		t.Fatal("Serve() error = nil, want error")
	}
	if !errors.Is(err, expectedErr) && err.Error() != "tcp.New(example.com:8080): websocket listener creation failed" {
		t.Errorf("Serve() error = %v, want wrapped listener creation error", err)
	}
}

// Test SSL/TLS without mutual authentication

func TestNew_WithSSL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Key:      "",
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	fakeCertPool := x509.NewCertPool()
	fakeCert := tls.Certificate{}

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			t.Error("newTCPListener should not be called in New()")
			return nil, nil
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			t.Error("newWSListener should not be called in New()")
			return nil, nil
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			if key != "" {
				t.Error("certGenerator expected empty key when no key is configured")
			}
			return fakeCertPool, fakeCert, nil
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("newServer() with SSL error = %v", err)
	}
	if s == nil {
		t.Fatal("newServer() returned nil server")
	}
	if s.handle == nil {
		t.Error("newServer() did not set handler")
	}
}

func TestNew_SSLCertGenerationError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Key:      "",
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	certErr := errors.New("certificate generation failed")

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			t.Error("newTCPListener should not be called in New()")
			return nil, nil
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			t.Error("newWSListener should not be called in New()")
			return nil, nil
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			return nil, tls.Certificate{}, certErr
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err == nil {
		t.Fatal("newServer() error = nil, want error")
	}
	if s != nil {
		t.Error("newServer() should return nil server on cert generation error")
	}
}

// Test SSL/TLS with mutual authentication

func TestNew_WithSSLAndMTLS(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Key:      "testkey",
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	fakeCertPool := x509.NewCertPool()
	fakeCert := tls.Certificate{}

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			t.Error("newTCPListener should not be called in New()")
			return nil, nil
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			t.Error("newWSListener should not be called in New()")
			return nil, nil
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			// Key is salted by cfg.GetKey()
			if key == "" {
				t.Error("certGenerator expected non-empty key when key is configured")
			}
			return fakeCertPool, fakeCert, nil
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("newServer() with SSL and mTLS error = %v", err)
	}
	if s == nil {
		t.Fatal("newServer() returned nil server")
	}
	if s.handle == nil {
		t.Error("newServer() did not set handler")
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

// Test Close

func TestServer_Close_NoListener(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	s, err := New(ctx, cfg, handler)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Close before Serve - listener is nil
	if err := s.Close(); err != nil {
		t.Errorf("Close() with nil listener error = %v", err)
	}
}

func TestServer_Close_WithListener(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		Timeout:  time.Second,
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	fakeListener := &fakeListener{}
	listenerCreated := make(chan struct{})

	deps := &dependencies{
		newTCPListener: func(addr string, deps *config.Dependencies) (transport.Listener, error) {
			close(listenerCreated) // Signal that listener has been created
			return fakeListener, nil
		},
		newWSListener: func(ctx context.Context, addr string, secure bool) (transport.Listener, error) {
			t.Error("newWSListener should not be called for TCP protocol")
			return nil, nil
		},
		certGenerator: func(key string) (*x509.CertPool, tls.Certificate, error) {
			t.Error("certGenerator should not be called when SSL is false")
			return nil, tls.Certificate{}, nil
		},
	}

	s, err := newServer(ctx, cfg, handler, deps)
	if err != nil {
		t.Fatalf("newServer() error = %v, want nil", err)
	}

	// Start serving to initialize listener
	go s.Serve()

	// Wait for listener to be created
	select {
	case <-listenerCreated:
		// Good, listener was created
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for listener to be created")
	}

	// Close should close the listener
	if err := s.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Check that listener was closed
	fakeListener.mu.Lock()
	closed := fakeListener.closed
	fakeListener.mu.Unlock()

	if !closed {
		t.Error("Close() did not close the listener")
	}
}
