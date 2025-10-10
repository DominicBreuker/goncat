package server

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"net"
	"testing"
)

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

func TestNew_WithSSL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Key:      "",
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	s, err := New(ctx, cfg, handler)
	if err != nil {
		t.Fatalf("New() with SSL error = %v", err)
	}
	if s == nil {
		t.Fatal("New() returned nil server")
	}
	if s.handle == nil {
		t.Error("New() did not set handler")
	}
}

func TestNew_WithSSLAndKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "localhost",
		Port:     8080,
		SSL:      true,
		Key:      "testkey",
	}

	handler := func(conn net.Conn) error {
		return nil
	}

	s, err := New(ctx, cfg, handler)
	if err != nil {
		t.Fatalf("New() with SSL and key error = %v", err)
	}
	if s == nil {
		t.Fatal("New() returned nil server")
	}
	if s.handle == nil {
		t.Error("New() did not set handler")
	}
}

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
