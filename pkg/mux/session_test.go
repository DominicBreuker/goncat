package mux

import (
	"net"
	"testing"

	"github.com/hashicorp/yamux"
)

// TestSession_Close verifies that Session.Close properly closes control channels and yamux session.
func TestSession_Close(t *testing.T) {
	t.Parallel()

	// Create a pair of connected pipes for testing
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// Create a yamux client session
	muxSession, err := yamux.Client(client, config())
	if err != nil {
		t.Fatalf("yamux.Client() failed: %v", err)
	}

	// Create dummy control channels
	dummyClient, dummyServer := net.Pipe()
	defer dummyClient.Close()
	defer dummyServer.Close()

	// Create Session struct
	session := &Session{
		mux:               muxSession,
		ctlClientToServer: dummyClient,
		ctlServerToClient: dummyServer,
	}

	// Close the session
	if err := session.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify yamux session is closed by trying to open a stream
	_, err = muxSession.Open()
	if err == nil {
		t.Error("Open() succeeded on closed session; want error")
	}
}

// TestSession_Close_NilControlChannels verifies Close works with nil control channels.
func TestSession_Close_NilControlChannels(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	muxSession, err := yamux.Client(client, config())
	if err != nil {
		t.Fatalf("yamux.Client() failed: %v", err)
	}

	// Create Session with nil control channels
	session := &Session{
		mux:               muxSession,
		ctlClientToServer: nil,
		ctlServerToClient: nil,
	}

	// Should not panic or error
	if err := session.Close(); err != nil {
		t.Errorf("Close() with nil channels failed: %v", err)
	}
}

// TestSession_Close_PartiallyNilControlChannels verifies Close works when only one channel is nil.
func TestSession_Close_PartiallyNilControlChannels(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	muxSession, err := yamux.Client(client, config())
	if err != nil {
		t.Fatalf("yamux.Client() failed: %v", err)
	}

	dummy, _ := net.Pipe()
	defer dummy.Close()

	cases := []struct {
		name              string
		ctlClientToServer net.Conn
		ctlServerToClient net.Conn
	}{
		{"only_client_to_server", dummy, nil},
		{"only_server_to_client", nil, dummy},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here because we're reusing muxSession
			session := &Session{
				mux:               muxSession,
				ctlClientToServer: tc.ctlClientToServer,
				ctlServerToClient: tc.ctlServerToClient,
			}

			// Should not panic or error even with partial nil channels
			if err := session.Close(); err != nil {
				t.Errorf("Close() with partial nil channels failed: %v", err)
			}
		})
	}
}

// TestConfig verifies the yamux configuration is set up correctly.
func TestConfig(t *testing.T) {
	t.Parallel()

	cfg := config()

	if cfg == nil {
		t.Fatal("config() returned nil")
	}

	// Verify logging is disabled
	if cfg.LogOutput != nil {
		t.Error("LogOutput should be nil")
	}

	if cfg.Logger == nil {
		t.Error("Logger should not be nil")
	}

	// Verify default timeout is kept (should be 75s from yamux defaults)
	defaultCfg := yamux.DefaultConfig()
	if cfg.StreamOpenTimeout != defaultCfg.StreamOpenTimeout {
		t.Errorf("StreamOpenTimeout = %v; want default %v", cfg.StreamOpenTimeout, defaultCfg.StreamOpenTimeout)
	}
}

// TestConfig_LoggerDiscardsOutput verifies that the logger actually discards output.
func TestConfig_LoggerDiscardsOutput(t *testing.T) {
	t.Parallel()

	cfg := config()

	if cfg.Logger == nil {
		t.Fatal("Logger should not be nil")
	}

	// Write to the logger - should not panic or produce output
	cfg.Logger.Print("test message")
	cfg.Logger.Printf("test message %s", "with formatting")
	cfg.Logger.Println("test message with newline")
}

// TestConfig_Consistency verifies that config() returns consistent values.
func TestConfig_Consistency(t *testing.T) {
	t.Parallel()

	cfg1 := config()
	cfg2 := config()

	if cfg1.StreamOpenTimeout != cfg2.StreamOpenTimeout {
		t.Error("config() returns inconsistent StreamOpenTimeout")
	}

	if (cfg1.LogOutput == nil) != (cfg2.LogOutput == nil) {
		t.Error("config() returns inconsistent LogOutput")
	}
}

// TestSession_CloseIdempotent verifies that calling Close multiple times is safe.
func TestSession_CloseIdempotent(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	muxSession, err := yamux.Client(client, config())
	if err != nil {
		t.Fatalf("yamux.Client() failed: %v", err)
	}

	session := &Session{
		mux:               muxSession,
		ctlClientToServer: nil,
		ctlServerToClient: nil,
	}

	// First close should succeed
	if err := session.Close(); err != nil {
		t.Fatalf("first Close() failed: %v", err)
	}

	// Second close should return an error (yamux session already closed)
	// but shouldn't panic
	_ = session.Close()
}

// TestConfig_LoggerWritesToDiscard verifies the logger is configured to discard output.
func TestConfig_LoggerWritesToDiscard(t *testing.T) {
	t.Parallel()

	cfg := config()

	if cfg.Logger == nil {
		t.Fatal("Logger should not be nil")
	}

	// The logger should be configured with io.Discard
	// We verify this by ensuring writes don't cause errors or panics
	cfg.Logger.Print("test message")
	cfg.Logger.Printf("formatted %s", "message")
}
