package integration

import (
	"context"
	"dominicbreuker/goncat/mocks"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/test/helpers"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLocalPortForwarding simulates a complete local port forwarding scenario
// with mocked network and demonstrates full end-to-end data flow through the tunnel.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345' -L 8000:127.0.0.1:9000" (master listening with local port forwarding)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
//
// With an additional mock server on the slave side at 127.0.0.1:9000 that responds with unique data,
// and a mock client on the master side connecting to the forwarded port 8000.
func TestLocalPortForwarding(t *testing.T) {
	// Create mock network for TCP connections
	mockNet := mocks.NewMockTCPNetwork()

	// Create mock stdio for master and slave (not used in this test but required for setup)
	masterStdio := mocks.NewMockStdio()
	slaveStdio := mocks.NewMockStdio()
	defer masterStdio.Close()
	defer slaveStdio.Close()

	// Setup mock "remote server" on slave side (this would be the server at 127.0.0.1:9000)
	// This server will respond with unique data when contacted
	remoteServerAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9000")
	if err != nil {
		t.Fatalf("Failed to resolve remote server address: %v", err)
	}

	remoteServerListener, err := mockNet.ListenTCP("tcp", remoteServerAddr)
	if err != nil {
		t.Fatalf("Failed to create remote server listener: %v", err)
	}
	defer remoteServerListener.Close()

	// Start mock remote server in a goroutine
	remoteServerStarted := make(chan struct{})
	go func() {
		close(remoteServerStarted)
		for {
			conn, err := remoteServerListener.Accept()
			if err != nil {
				return // listener closed
			}
			go func(c net.Conn) {
				defer c.Close()
				// Read request data
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil && err != io.EOF {
					t.Logf("Remote server read error: %v", err)
					return
				}
				request := string(buf[:n])
				// Respond with unique data that includes the request
				response := fmt.Sprintf("REMOTE_SERVER_RESPONSE: You sent '%s'", strings.TrimSpace(request))
				c.Write([]byte(response))
			}(conn)
		}
	}()

	// Wait for remote server to start
	<-remoteServerStarted
	// Note: The channel close is enough to signal the server is ready

	// Setup master dependencies (network + stdio)
	// TCPDialer and TCPListener are used for all TCP operations including port forwarding
	masterDeps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
		Stdin:       func() io.Reader { return masterStdio.GetStdin() },
		Stdout:      func() io.Writer { return masterStdio.GetStdout() },
	}

	// Setup slave dependencies (network + stdio)
	slaveDeps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
		Stdin:       func() io.Reader { return slaveStdio.GetStdin() },
		Stdout:      func() io.Writer { return slaveStdio.GetStdout() },
	}

	// Master configuration with local port forwarding
	// Simulates "master listen 'tcp://*:12345' -L 8000:127.0.0.1:9000"
	masterSharedCfg := helpers.DefaultSharedConfig(masterDeps)
	masterCfg := helpers.DefaultMasterConfig()
	masterCfg.LocalPortForwarding = []*config.LocalPortForwardingCfg{
		{
			LocalHost:  "127.0.0.1",
			LocalPort:  8000,
			RemoteHost: "127.0.0.1",
			RemotePort: 9000,
		},
	}

	// Slave configuration - simulates "slave connect tcp://127.0.0.1:12345"
	slaveSharedCfg := helpers.DefaultSharedConfig(slaveDeps)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server using entrypoint (listens for connections and sets up port forwarding)
	go func() {
		if err := entrypoint.MasterListen(ctx, masterSharedCfg, masterCfg); err != nil {
			// Context cancellation is expected
			select {
			case <-ctx.Done():
				masterErr <- nil
			default:
				masterErr <- err
			}
			return
		}
		masterErr <- nil
	}()

	// Wait for master to start listening
	if err := mockNet.WaitForListener("127.0.0.1:12345", 2000); err != nil {
		t.Fatalf("Master failed to start listening: %v", err)
	}

	// Start slave using entrypoint (connects to master)
	go func() {
		if err := entrypoint.SlaveConnect(ctx, slaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	// Wait for the forwarded port to be available (local port forwarding listener on master side)
	if err := mockNet.WaitForListener("127.0.0.1:8000", 2000); err != nil {
		t.Fatalf("Forwarded port failed to start listening: %v", err)
	}

	// Now test the port forwarding by connecting a mock client to the forwarded port
	// This client connects to 127.0.0.1:8000 on the master side
	forwardedPortAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8000")
	if err != nil {
		t.Fatalf("Failed to resolve forwarded port address: %v", err)
	}

	// Use a waitgroup to ensure the client goroutine completes
	var clientWg sync.WaitGroup
	clientWg.Add(1)

	// Track client test results
	clientSuccess := false
	var clientErr error
	var clientResponse string

	go func() {
		defer clientWg.Done()

		// Connect to the forwarded port
		clientConn, err := mockNet.DialTCP("tcp", nil, forwardedPortAddr)
		if err != nil {
			clientErr = fmt.Errorf("failed to connect to forwarded port: %v", err)
			return
		}
		defer clientConn.Close()

		// Send test data through the tunnel
		testData := "Hello through tunnel!"
		_, err = clientConn.Write([]byte(testData))
		if err != nil {
			clientErr = fmt.Errorf("failed to write to forwarded port: %v", err)
			return
		}

		// Read response from the remote server (should come through the tunnel)
		buf := make([]byte, 1024)
		clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := clientConn.Read(buf)
		if err != nil && err != io.EOF {
			clientErr = fmt.Errorf("failed to read response: %v", err)
			return
		}

		clientResponse = string(buf[:n])
		clientSuccess = true
	}()

	// Wait for client to complete
	clientWg.Wait()

	// Verify the client test results
	if clientErr != nil {
		t.Errorf("Client error: %v", clientErr)
	}

	if !clientSuccess {
		t.Fatal("Client failed to complete successfully")
	}

	// Verify the response contains both the expected prefix and the sent data
	expectedPrefix := "REMOTE_SERVER_RESPONSE:"
	expectedContent := "Hello through tunnel!"
	if !strings.Contains(clientResponse, expectedPrefix) {
		t.Errorf("Expected response to contain '%s', got: %q", expectedPrefix, clientResponse)
	}
	if !strings.Contains(clientResponse, expectedContent) {
		t.Errorf("Expected response to contain sent data '%s', got: %q", expectedContent, clientResponse)
	}

	t.Logf("✓ Port forwarding test successful! Response: %q", clientResponse)

	// Test multiple connections to ensure port forwarding is stable
	for i := 0; i < 3; i++ {
		clientWg.Add(1)
		go func(iteration int) {
			defer clientWg.Done()

			clientConn, err := mockNet.DialTCP("tcp", nil, forwardedPortAddr)
			if err != nil {
				t.Errorf("Iteration %d: failed to connect: %v", iteration, err)
				return
			}
			defer clientConn.Close()

			testData := fmt.Sprintf("Message %d", iteration)
			clientConn.Write([]byte(testData))

			buf := make([]byte, 1024)
			clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := clientConn.Read(buf)
			if err != nil && err != io.EOF {
				t.Errorf("Iteration %d: failed to read: %v", iteration, err)
				return
			}

			response := string(buf[:n])
			if !strings.Contains(response, testData) {
				t.Errorf("Iteration %d: expected response to contain '%s', got: %q", iteration, testData, response)
			} else {
				t.Logf("✓ Iteration %d successful! Response: %q", iteration, response)
			}
		}(i)
	}

	// Wait for all iterations to complete
	clientWg.Wait()

	// Cleanup
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Check for errors (non-blocking)
	select {
	case err := <-masterErr:
		if err != nil {
			t.Logf("Master completed with: %v", err)
		}
	default:
		t.Log("Master still running (expected)")
	}

	select {
	case err := <-slaveErr:
		if err != nil {
			t.Logf("Slave completed with: %v", err)
		}
	default:
		t.Log("Slave still running (expected)")
	}
}
