package portfwd

import (
	"context"
	mocks_tcp "dominicbreuker/goncat/mocks/tcp"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/test/helpers"
	"strings"
	"testing"
	"time"
)

// TestRemotePortForwarding simulates a complete remote port forwarding scenario
// with mocked network and demonstrates full end-to-end data flow through the tunnel.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345' -R 8000:127.0.0.1:9000" (master listening with remote port forwarding)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
//
// With remote port forwarding, the slave binds port 8000, and connections to it are tunneled
// to the master side, which then connects to 127.0.0.1:9000 (from the master's perspective).
// This is the reverse of local port forwarding.
func TestRemotePortForwarding(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Setup mock "remote server" on master side (this would be the server at 127.0.0.1:9000)
	// This server will respond with unique data when contacted
	// Start the reusable mock echo server using the mock network's ListenTCP
	srv, err := mocks_tcp.NewServer(setup.TCPNetwork.ListenTCP, "tcp", "127.0.0.1:9000", "REMOTE_SERVER_RESPONSE: ")
	if err != nil {
		t.Fatalf("Failed to start remote server: %v", err)
	}
	defer srv.Close()

	// Configure master with remote port forwarding
	// Simulates "master listen 'tcp://*:12345' -R 127.0.0.1:8000:127.0.0.1:9000"
	// This tells the slave to bind port 8000 on 127.0.0.1 and forward connections to master's 127.0.0.1:9000
	// Use ParseRemotePortForwardingSpecs to properly initialize the config
	// Format: [slave_host]:[slave_port]:[master_host]:[master_port]
	setup.MasterCfg.ParseRemotePortForwardingSpecs([]string{"127.0.0.1:8000:127.0.0.1:9000"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server using entrypoint (listens for connections and sets up port forwarding)
	go func() {
		if err := entrypoint.MasterListen(ctx, setup.MasterSharedCfg, setup.MasterCfg); err != nil {
			masterErr <- err
			return
		}
		masterErr <- nil
	}()

	// Wait for master listener
	var lMaster *mocks_tcp.MockTCPListener
	if lMaster, err = setup.TCPNetwork.WaitForListener("127.0.0.1:12345", 2000); err != nil {
		t.Fatalf("Master failed to start listening: %v", err)
	}
	t.Logf("Master has started listening on %s", lMaster.Addr().String())

	// Start slave using entrypoint (connects to master)
	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	var cSlaveToMaster *mocks_tcp.MockTCPConn
	if cSlaveToMaster, err = lMaster.WaitForNewConnection(2000); err != nil {
		t.Fatalf("Slave failed to connect to master: %v", err)
	}
	t.Logf("Slave is connected to master: %s->%s", cSlaveToMaster.RemoteAddr().String(), cSlaveToMaster.LocalAddr().String())

	// Wait for forwarding listener and its target echo server
	var lRemote *mocks_tcp.MockTCPListener
	if lRemote, err = setup.TCPNetwork.WaitForListener("127.0.0.1:8000", 2000); err != nil {
		t.Fatalf("Forwarded port failed to start listening: %v", err)
	}
	t.Logf("Slave has started listening on %s for remote port forwarding", lRemote.Addr().String())

	var lDestination *mocks_tcp.MockTCPListener
	if lDestination, err = setup.TCPNetwork.WaitForListener("127.0.0.1:9000", 2000); err != nil {
		t.Fatalf("Destination server failed to start listening: %v", err)
	}
	t.Logf("Destination server has started listening on %v", lDestination.Addr().String())

	client, err := mocks_tcp.NewClient(setup.TCPNetwork.DialTCP, "tcp", "127.0.0.1:8000")
	if err != nil {
		t.Fatalf("failed to connect to forwarded port: %v", err)
	}
	defer client.Close()

	var cClientToRelay *mocks_tcp.MockTCPConn
	if cClientToRelay, err = lRemote.WaitForNewConnection(2000); err != nil {
		t.Fatalf("Client failed to connect to forwarded port: %v", err)
	}
	t.Logf("Client connected to forwarded port: %s->%s", cClientToRelay.RemoteAddr().String(), cClientToRelay.LocalAddr().String())

	var cRelayToDestination *mocks_tcp.MockTCPConn
	if cRelayToDestination, err = lDestination.WaitForNewConnection(2000); err != nil {
		t.Fatalf("Relay failed to connect to destination server: %v", err)
	}
	t.Logf("Relay connected to destination server: %s->%s", cRelayToDestination.RemoteAddr().String(), cRelayToDestination.LocalAddr().String())

	testData := "Hello through reverse tunnel!"
	if err := client.WriteLine(testData); err != nil {
		t.Fatalf("failed to write to forwarded port: %v", err)
	}

	// required, the piping through the relay is not immediate
	time.Sleep(100 * time.Millisecond)

	resp, err := client.ReadLine()
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Verify the response contains both the expected prefix and the sent data
	expectedPrefix := "REMOTE_SERVER_RESPONSE:"
	expectedContent := "Hello through reverse tunnel!"
	if !strings.Contains(resp, expectedPrefix) {
		t.Errorf("Expected response to contain '%s', got: %q", expectedPrefix, resp)
	}
	if !strings.Contains(resp, expectedContent) {
		t.Errorf("Expected response to contain sent data '%s', got: %q", expectedContent, resp)
	}

	t.Logf("✓ Remote port forwarding test successful! Response: %q", resp)

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
