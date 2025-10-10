package client

import (
	"context"
	"crypto/x509"
	"dominicbreuker/goncat/pkg/config"
	"testing"
)

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
