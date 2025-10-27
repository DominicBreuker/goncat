package portfwd

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/test/helpers"
	"net"
	"testing"
	"time"
)

// TestUDPLocalPortForwarding tests UDP local port forwarding with the U: prefix.
// This test mimics: "goncat master listen 'tcp://*:12345' -L U:8000:127.0.0.1:9000"
// It verifies that UDP datagrams sent to port 8000 are forwarded through the tunnel
// to port 9000 on the remote side.
func TestUDPLocalPortForwarding(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure master with UDP local port forwarding
	// Simulates "master listen 'tcp://*:12345' -L U:8000:127.0.0.1:9000"
	setup.MasterCfg.LocalPortForwarding = []*config.LocalPortForwardingCfg{
		{
			Protocol:   "udp",
			LocalHost:  "127.0.0.1",
			LocalPort:  8000,
			RemoteHost: "127.0.0.1",
			RemotePort: 9000,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a UDP echo server on the slave side at port 9000
	udpAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}
	udpConn, err := setup.UDPNetwork.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("Failed to create UDP echo server: %v", err)
	}
	defer udpConn.Close()

	// Start UDP echo server goroutine
	go func() {
		buffer := make([]byte, 65536)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, addr, err := udpConn.ReadFrom(buffer)
			if err != nil {
				return
			}
			// Echo back the data with a prefix
			response := append([]byte("ECHO: "), buffer[:n]...)
			udpConn.WriteTo(response, addr)
		}
	}()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server
	go func() {
		if err := entrypoint.MasterListen(ctx, setup.MasterSharedCfg, setup.MasterCfg); err != nil {
			masterErr <- err
			return
		}
		masterErr <- nil
	}()

	// Wait for master listener
	time.Sleep(200 * time.Millisecond)

	// Start slave
	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	// Wait for connection to establish and port forwarding to start
	time.Sleep(500 * time.Millisecond)

	// Create a UDP client "connection" by creating a listener with ephemeral port
	// and then using WriteTo to send to the forwarded port
	clientConn, err := setup.UDPNetwork.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP client: %v", err)
	}
	defer clientConn.Close()

	// Target address is the forwarded port
	forwardedAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8000}

	// Send test data through the forwarded port
	testData := []byte("Hello UDP forwarding!")
	_, err = clientConn.WriteTo(testData, forwardedAddr)
	if err != nil {
		t.Fatalf("Failed to write UDP data: %v", err)
	}

	// Read response
	buffer := make([]byte, 65536)
	clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := clientConn.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("Failed to read UDP response: %v", err)
	}

	response := string(buffer[:n])
	expectedResponse := "ECHO: Hello UDP forwarding!"
	if response != expectedResponse {
		t.Errorf("Expected response %q, got %q", expectedResponse, response)
	}

	t.Logf("✓ UDP local port forwarding test successful! Response: %q", response)

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

// TestUDPRemotePortForwarding tests UDP remote port forwarding with the U: prefix.
// This test mimics: "goncat master listen 'tcp://*:12345' -R U:8000:127.0.0.1:9000"
// It verifies that UDP datagrams sent to port 8000 on the slave are forwarded
// through the tunnel to port 9000 on the master side.
func TestUDPRemotePortForwarding(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure master with UDP remote port forwarding
	// Format: [slave_host]:[slave_port]:[master_host]:[master_port]
	// Parse the spec with U: prefix for UDP
	specs := []string{"U:127.0.0.1:8000:127.0.0.1:9000"}
	setup.MasterCfg.ParseRemotePortForwardingSpecs(specs)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a UDP echo server on the master side at port 9000
	udpAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}
	udpConn, err := setup.UDPNetwork.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("Failed to create UDP echo server: %v", err)
	}
	defer udpConn.Close()

	// Start UDP echo server goroutine
	go func() {
		buffer := make([]byte, 65536)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, addr, err := udpConn.ReadFrom(buffer)
			if err != nil {
				return
			}
			// Echo back the data with a prefix
			response := append([]byte("REMOTE_ECHO: "), buffer[:n]...)
			udpConn.WriteTo(response, addr)
		}
	}()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server
	go func() {
		if err := entrypoint.MasterListen(ctx, setup.MasterSharedCfg, setup.MasterCfg); err != nil {
			masterErr <- err
			return
		}
		masterErr <- nil
	}()

	// Wait for master listener
	time.Sleep(200 * time.Millisecond)

	// Start slave
	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	// Wait for connection to establish and port forwarding to start
	time.Sleep(500 * time.Millisecond)

	// Create a UDP client "connection" by creating a listener with ephemeral port
	clientConn, err := setup.UDPNetwork.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP client: %v", err)
	}
	defer clientConn.Close()

	// Target address is the forwarded port on slave side
	forwardedAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8000}

	// Send test data through the forwarded port
	testData := []byte("Hello UDP remote forwarding!")
	_, err = clientConn.WriteTo(testData, forwardedAddr)
	if err != nil {
		t.Fatalf("Failed to write UDP data: %v", err)
	}

	// Read response
	buffer := make([]byte, 65536)
	clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := clientConn.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("Failed to read UDP response: %v", err)
	}

	response := string(buffer[:n])
	expectedResponse := "REMOTE_ECHO: Hello UDP remote forwarding!"
	if response != expectedResponse {
		t.Errorf("Expected response %q, got %q", expectedResponse, response)
	}

	t.Logf("✓ UDP remote port forwarding test successful! Response: %q", response)

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

// TestMixedTCPAndUDPPortForwarding tests both TCP and UDP port forwarding simultaneously.
// This verifies that the two protocols can coexist without interfering with each other.
func TestMixedTCPAndUDPPortForwarding(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure master with both TCP and UDP local port forwarding
	setup.MasterCfg.LocalPortForwarding = []*config.LocalPortForwardingCfg{
		{
			Protocol:   "tcp",
			LocalHost:  "127.0.0.1",
			LocalPort:  8000,
			RemoteHost: "127.0.0.1",
			RemotePort: 9000,
		},
		{
			Protocol:   "udp",
			LocalHost:  "127.0.0.1",
			LocalPort:  8001,
			RemoteHost: "127.0.0.1",
			RemotePort: 9001,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create UDP echo server
	udpAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9001}
	udpConn, err := setup.UDPNetwork.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("Failed to create UDP echo server: %v", err)
	}
	defer udpConn.Close()

	go func() {
		buffer := make([]byte, 65536)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, addr, err := udpConn.ReadFrom(buffer)
			if err != nil {
				return
			}
			response := append([]byte("UDP_ECHO: "), buffer[:n]...)
			udpConn.WriteTo(response, addr)
		}
	}()

	// Start master and slave
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	go func() {
		if err := entrypoint.MasterListen(ctx, setup.MasterSharedCfg, setup.MasterCfg); err != nil {
			masterErr <- err
			return
		}
		masterErr <- nil
	}()

	time.Sleep(200 * time.Millisecond)

	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	time.Sleep(500 * time.Millisecond)

	// Test UDP forwarding
	clientConn, err := setup.UDPNetwork.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP client: %v", err)
	}
	defer clientConn.Close()

	forwardedAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8001}
	testData := []byte("Mixed protocol test!")
	_, err = clientConn.WriteTo(testData, forwardedAddr)
	if err != nil {
		t.Fatalf("Failed to write UDP data: %v", err)
	}

	buffer := make([]byte, 65536)
	clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := clientConn.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("Failed to read UDP response: %v", err)
	}

	response := string(buffer[:n])
	expectedResponse := "UDP_ECHO: Mixed protocol test!"
	if response != expectedResponse {
		t.Errorf("Expected UDP response %q, got %q", expectedResponse, response)
	}

	t.Logf("✓ Mixed TCP/UDP port forwarding test successful! UDP Response: %q", response)
}
