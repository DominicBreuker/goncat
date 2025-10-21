package portfwd

import (
	"context"
	mocks_tcp "dominicbreuker/goncat/mocks/tcp"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/test/helpers"
	"strings"
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
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure master with local port forwarding
	// Simulates "master listen 'tcp://*:12345' -L 8000:127.0.0.1:9000"
	setup.MasterCfg.LocalPortForwarding = []*config.LocalPortForwardingCfg{
		{
			LocalHost:  "127.0.0.1",
			LocalPort:  8000,
			RemoteHost: "127.0.0.1",
			RemotePort: 9000,
		},
	}

	// Setup mock "remote server" on slave side (this would be the server at 127.0.0.1:9000)
	// This server will respond with unique data when contacted
	// Start the reusable mock echo server using the mock network's ListenTCP
	srv, err := mocks_tcp.NewServer(setup.TCPNetwork.ListenTCP, "tcp", "127.0.0.1:9000", "REMOTE_SERVER_RESPONSE: ")
	if err != nil {
		t.Fatalf("Failed to start remote server: %v", err)
	}
	defer srv.Close()

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
	var lLocal *mocks_tcp.MockTCPListener
	if lLocal, err = setup.TCPNetwork.WaitForListener("127.0.0.1:8000", 2000); err != nil {
		t.Fatalf("Forwarded port failed to start listening: %v", err)
	}
	t.Logf("Master has started listening on %s for port forwarding", lLocal.Addr().String())

	var lRemote *mocks_tcp.MockTCPListener
	if lRemote, err = setup.TCPNetwork.WaitForListener("127.0.0.1:9000", 2000); err != nil {
		t.Fatalf("Echo server failed to start listening: %v", err)
	}
	t.Logf("Remote server has started listening on %v", lRemote.Addr().String())

	client, err := mocks_tcp.NewClient(setup.TCPNetwork.DialTCP, "tcp", "127.0.0.1:8000")
	if err != nil {
		t.Fatalf("failed to connect to forwarded port: %v", err)
	}
	defer client.Close()

	var cClientToRelay *mocks_tcp.MockTCPConn
	if cClientToRelay, err = lLocal.WaitForNewConnection(2000); err != nil {
		t.Fatalf("Local client failed to connect to forwarded port: %v", err)
	}
	t.Logf("Local TCP client connected to forwarded port: %s->%s", cClientToRelay.RemoteAddr().String(), cClientToRelay.LocalAddr().String())

	var cRelayToRemote *mocks_tcp.MockTCPConn
	if cRelayToRemote, err = lRemote.WaitForNewConnection(2000); err != nil {
		t.Fatalf("Relay failed to connect to remote server: %v", err)
	}
	t.Logf("Relay connectec to remote server: %s->%s", cRelayToRemote.RemoteAddr().String(), cRelayToRemote.LocalAddr().String())

	testData := "Hello through tunnel!"
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
	expectedContent := "Hello through tunnel!"
	if !strings.Contains(resp, expectedPrefix) {
		t.Errorf("Expected response to contain '%s', got: %q", expectedPrefix, resp)
	}
	if !strings.Contains(resp, expectedContent) {
		t.Errorf("Expected response to contain sent data '%s', got: %q", expectedContent, resp)
	}

	t.Logf("✓ Port forwarding test successful! Response: %q", resp)

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
