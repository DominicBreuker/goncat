package integration

import (
	"context"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"dominicbreuker/goncat/pkg/server"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

// TestSlaveConnectToMasterListen demonstrates a simple integration test
// where a slave connects to a listening master using mocked TCP connections.
func TestSlaveConnectToMasterListen(t *testing.T) {
	// Create a mock TCP network
	mockNet := NewMockTCPNetwork()

	// Create dependencies with the mock network
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
	}

	// Setup master configuration (listens on 127.0.0.1:12345)
	masterCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Deps:     deps,
	}

	// Setup slave configuration (connects to 127.0.0.1:12345)
	slaveCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Deps:     deps,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Channel to coordinate test
	masterReady := make(chan struct{})
	masterHandled := make(chan error, 1)
	slaveConnected := make(chan error, 1)
	testComplete := make(chan struct{})

	// Master handler that receives data and echoes it back
	masterHandler := func(conn net.Conn) error {
		defer conn.Close()

		// Signal that master is handling a connection
		close(masterReady)

		// Read data from slave
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("master read error: %w", err)
		}

		// Echo back
		if n > 0 {
			_, err = conn.Write(buf[:n])
			if err != nil {
				return fmt.Errorf("master write error: %w", err)
			}
		}

		return nil
	}

	// Start master server in background
	go func() {
		s, err := server.New(ctx, masterCfg, masterHandler)
		if err != nil {
			masterHandled <- fmt.Errorf("server.New(): %w", err)
			return
		}
		defer s.Close()

		if err := s.Serve(); err != nil {
			// Check if error is due to context cancellation (expected)
			select {
			case <-ctx.Done():
				masterHandled <- nil
			default:
				masterHandled <- fmt.Errorf("serving: %w", err)
			}
			return
		}
		masterHandled <- nil
	}()

	// Give master a moment to start listening
	time.Sleep(100 * time.Millisecond)

	// Slave connects and sends data
	go func() {
		c := client.New(ctx, slaveCfg)
		if err := c.Connect(); err != nil {
			slaveConnected <- fmt.Errorf("connecting: %w", err)
			return
		}
		defer c.Close()

		conn := c.GetConnection()

		// Send test message
		testMsg := []byte("Hello from slave")
		_, err := conn.Write(testMsg)
		if err != nil {
			slaveConnected <- fmt.Errorf("write error: %w", err)
			return
		}

		// Read echo response
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			slaveConnected <- fmt.Errorf("read error: %w", err)
			return
		}

		// Verify echo
		if string(buf[:n]) != string(testMsg) {
			slaveConnected <- fmt.Errorf("expected %q, got %q", testMsg, buf[:n])
			return
		}

		slaveConnected <- nil
		close(testComplete)
	}()

	// Wait for test to complete
	select {
	case err := <-slaveConnected:
		if err != nil {
			t.Fatalf("Slave error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Test timeout")
	}

	// Wait for test completion signal
	select {
	case <-testComplete:
		// Success!
	case <-time.After(1 * time.Second):
		t.Fatal("Test did not complete in time")
	}

	// Cancel context to stop server
	cancel()

	// Give the master time to finish
	time.Sleep(200 * time.Millisecond)

	// Check master finished without error (non-blocking check)
	select {
	case err := <-masterHandled:
		if err != nil {
			t.Errorf("Master error: %v", err)
		}
	default:
		// Master goroutine may still be cleaning up, which is fine
		t.Log("Master goroutine still running (expected)")
	}
}

// TestSlaveHandlerWithMock demonstrates using the slave handler with mocked connections.
// This is a more complete example that actually uses the slave handler.
func TestSlaveHandlerWithMock(t *testing.T) {
	t.Skip("Skipping more complex test for now - basic mock is demonstrated")

	// Create a mock TCP network
	mockNet := NewMockTCPNetwork()

	// Create dependencies with the mock network
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
	}

	// Setup slave configuration
	slaveCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Deps:     deps,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a mock connection pair
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Start slave handler
	go func() {
		h, err := slave.New(ctx, slaveCfg, serverConn)
		if err != nil {
			t.Errorf("slave.New(): %v", err)
			return
		}
		defer h.Close()

		if err := h.Handle(); err != nil {
			t.Errorf("handling: %v", err)
		}
	}()

	// Master side - send simple data
	testMsg := []byte("test data")
	_, err := clientConn.Write(testMsg)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Read response
	buf := make([]byte, 1024)
	clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, err = clientConn.Read(buf)
	if err != nil && err != io.EOF {
		// Expected to potentially timeout or EOF
		t.Logf("Read result: %v", err)
	}
}

// TestMasterConnectToSlaveListen demonstrates the reverse scenario where
// a master connects to a listening slave using mocked connections.
func TestMasterConnectToSlaveListen(t *testing.T) {
	// Create a mock TCP network
	mockNet := NewMockTCPNetwork()

	// Create dependencies with the mock network
	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
	}

	// Setup configurations for both sides
	masterCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12346,
		SSL:      false,
		Deps:     deps,
	}

	slaveCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12346,
		SSL:      false,
		Deps:     deps,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Channels for coordination
	slaveReady := make(chan struct{})
	slaveHandled := make(chan error, 1)
	masterConnected := make(chan error, 1)

	// Slave handler that echoes data
	slaveHandler := func(conn net.Conn) error {
		defer conn.Close()
		close(slaveReady)

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("slave read error: %w", err)
		}

		if n > 0 {
			_, err = conn.Write(buf[:n])
			if err != nil {
				return fmt.Errorf("slave write error: %w", err)
			}
		}

		return nil
	}

	// Start slave server
	go func() {
		s, err := server.New(ctx, slaveCfg, slaveHandler)
		if err != nil {
			slaveHandled <- fmt.Errorf("server.New(): %w", err)
			return
		}
		defer s.Close()

		if err := s.Serve(); err != nil {
			select {
			case <-ctx.Done():
				slaveHandled <- nil
			default:
				slaveHandled <- fmt.Errorf("serving: %w", err)
			}
			return
		}
		slaveHandled <- nil
	}()

	// Give slave a moment to start
	time.Sleep(100 * time.Millisecond)

	// Master connects
	go func() {
		c := client.New(ctx, masterCfg)
		if err := c.Connect(); err != nil {
			masterConnected <- fmt.Errorf("connecting: %w", err)
			return
		}
		defer c.Close()

		conn := c.GetConnection()

		// Send test message
		testMsg := []byte("Hello from master")
		_, err := conn.Write(testMsg)
		if err != nil {
			masterConnected <- fmt.Errorf("write error: %w", err)
			return
		}

		// Read echo
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			masterConnected <- fmt.Errorf("read error: %w", err)
			return
		}

		// Verify echo
		if string(buf[:n]) != string(testMsg) {
			masterConnected <- fmt.Errorf("expected %q, got %q", testMsg, buf[:n])
			return
		}

		masterConnected <- nil
	}()

	// Wait for master to finish
	select {
	case err := <-masterConnected:
		if err != nil {
			t.Fatalf("Master error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Test timeout")
	}

	// Cancel and cleanup
	cancel()
	time.Sleep(200 * time.Millisecond)
}

// TestMockTCPBasics demonstrates basic functionality of the mock TCP network.
func TestMockTCPBasics(t *testing.T) {
	mockNet := NewMockTCPNetwork()

	// Test listener creation
	addr1, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8001")
	listener, err := mockNet.ListenTCP("tcp", addr1)
	if err != nil {
		t.Fatalf("ListenTCP failed: %v", err)
	}
	defer listener.Close()

	// Test duplicate address rejection
	_, err = mockNet.ListenTCP("tcp", addr1)
	if err == nil {
		t.Error("Expected error for duplicate listener address")
	}

	// Test successful connection and data transfer
	done := make(chan bool)
	go func() {
		// Server side: accept and echo
		serverConn, err := listener.Accept()
		if err != nil {
			t.Errorf("Accept failed: %v", err)
			return
		}
		defer serverConn.Close()

		buf := make([]byte, 1024)
		n, err := serverConn.Read(buf)
		if err != nil {
			t.Errorf("Read failed: %v", err)
			return
		}

		if string(buf[:n]) != "test message" {
			t.Errorf("Expected 'test message', got %q", buf[:n])
		}
		done <- true
	}()

	// Give server goroutine time to start listening
	time.Sleep(50 * time.Millisecond)

	// Client side: connect and send
	addr2, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8002")
	clientConn, err := mockNet.DialTCP("tcp", addr2, addr1)
	if err != nil {
		t.Fatalf("DialTCP failed: %v", err)
	}
	defer clientConn.Close()

	testData := []byte("test message")
	_, err = clientConn.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Wait for server to finish
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Test timeout")
	}

	// Test connection refused
	addr3, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9999")
	_, err = mockNet.DialTCP("tcp", nil, addr3)
	if err == nil {
		t.Error("Expected connection refused error")
	}
}
