package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"encoding/gob"
	"errors"
	"net"
	"testing"
	"time"

	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/mux/msg"
)

func TestNewUDPRelay_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	// Create a pipe for the remote connection
	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	if relay == nil {
		t.Fatal("NewUDPRelay() returned nil")
	}

	if relay.ConnLocal == nil {
		t.Error("Local UDP connection not created")
	}

	if relay.ConnRemote != serverConn {
		t.Error("Remote connection not set correctly")
	}
}

func TestNewUDPRelay_GetChannelError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	expectedErr := errors.New("channel error")
	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return nil, expectedErr
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err == nil {
		if relay != nil {
			relay.Close()
		}
		t.Error("Expected error when GetOneChannel fails, got nil")
	}
}

func TestUDPRelay_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}

	// First close should succeed
	err = relay.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify isClosed flag is set
	if !relay.isClosed {
		t.Error("isClosed flag not set after Close()")
	}

	// Second close should also work
	_ = relay.Close()
}

func TestUDPRelay_LogError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	// LogError should not panic
	relay.LogError("test error message")
	relay.LogError("test with args: %s %d", "string", 123)
}

func TestUDPRelay_SendToDst_WhenClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}

	// Close the relay first
	relay.Close()

	// Attempting to send should fail
	err = relay.sendToDst("127.0.0.1", 8080, []byte("test"))
	if err == nil {
		t.Error("Expected error when sending to closed relay, got nil")
	}
}

func TestUDPRelay_SendToDst_InvalidAddr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	// Use invalid address format
	err = relay.sendToDst("invalid address", -1, []byte("test"))
	if err == nil {
		t.Error("Expected error with invalid address, got nil")
	}
}

func TestUDPRelay_ContextPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration-style test in short mode")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, serverConn := net.Pipe()
	defer serverConn.Close()

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return serverConn, nil
		},
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err != nil {
		t.Fatalf("NewUDPRelay() returned error: %v", err)
	}
	defer relay.Close()

	if relay.ctx == nil {
		t.Error("Context not set in relay")
	}

	// Cancel context
	cancel()

	// Context in relay should be cancelled
	select {
	case <-relay.ctx.Done():
		// Success - context was cancelled
	default:
		t.Error("Relay context not cancelled when parent cancelled")
	}
}

// TestServeReturnsWhenRemoteCloses ensures Serve returns when the remote control conn closes.
func TestServeReturnsWhenRemoteCloses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-style test in short mode")
	}
	ctx := context.Background()

	// create a paired net.Pipe; one end will be given to relay, we close the other to simulate remote close
	rConn, wConn := net.Pipe()
	defer rConn.Close()
	// wConn will be closed by test

	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) { return rConn, nil },
	}

	relay, err := NewUDPRelay(ctx, sessCtl, log.NewLogger(false), nil)
	if err != nil {
		t.Fatalf("NewUDPRelay() error: %v", err)
	}
	defer relay.Close()

	done := make(chan error)
	go func() { done <- relay.Serve() }()

	// give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	// closing the writer side simulates remote closing the control channel
	wConn.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Serve did not return after remote close")
	}
}

// TestSendToDstWritesDatagram verifies sendToDst writes a UDP datagram to the bound PacketConn.
func TestSendToDstWritesDatagram(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-style test in short mode")
	}
	// create packet listener
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket: %v", err)
	}
	defer pc.Close()

	addr := pc.LocalAddr().(*net.UDPAddr)

	// provide a dummy session conn
	c1, _ := net.Pipe()
	defer c1.Close()
	sess := &fakeClientControlSession{getOneChannelFn: func() (net.Conn, error) { return c1, nil }}
	d := &config.Dependencies{PacketListener: func(network, address string) (net.PacketConn, error) { return pc, nil }}

	relay, err := NewUDPRelay(context.Background(), sess, log.NewLogger(false), d)
	if err != nil {
		t.Fatalf("NewUDPRelay: %v", err)
	}
	defer relay.Close()

	payload := []byte("hello")
	if err := relay.sendToDst(addr.IP.String(), addr.Port, payload); err != nil {
		t.Fatalf("sendToDst error: %v", err)
	}

	buf := make([]byte, 1024)
	pc.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if string(buf[:n]) != string(payload) {
		t.Fatalf("unexpected payload: %s", string(buf[:n]))
	}
}

// TestLocalToRemoteEncodesDatagram verifies that localToRemote encodes incoming UDP packets
// and writes them to the remote control connection.
func TestLocalToRemoteEncodesDatagram(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-style test in short mode")
	}
	// create pipe for remote conn where encoder writes
	rConn, wConn := net.Pipe()
	defer rConn.Close()
	defer wConn.Close()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket: %v", err)
	}
	defer pc.Close()

	addr := pc.LocalAddr().(*net.UDPAddr)

	sess := &fakeClientControlSession{getOneChannelFn: func() (net.Conn, error) { return rConn, nil }}
	d := &config.Dependencies{PacketListener: func(network, address string) (net.PacketConn, error) { return pc, nil }}

	relay, err := NewUDPRelay(context.Background(), sess, log.NewLogger(false), d)
	if err != nil {
		t.Fatalf("NewUDPRelay: %v", err)
	}
	defer relay.Close()

	// start only localToRemote which writes gob to the writer side
	go relay.localToRemote()

	// send a UDP packet to the packet listener
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("DialUDP: %v", err)
	}
	defer conn.Close()

	payload := []byte("ping")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}

	dec := gob.NewDecoder(wConn)
	var p msg.SocksDatagram
	wConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if err := dec.Decode(&p); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(p.Data) != string(payload) {
		t.Fatalf("unexpected data: %s", string(p.Data))
	}
}
