package connect

import (
	"bufio"
	"context"
	mocks_tcp "dominicbreuker/goncat/mocks/tcp"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/pkg/socks"
	"dominicbreuker/goncat/test/helpers"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSocksConnect simulates a complete SOCKS5 proxy scenario with mocked network.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345' -D 127.0.0.1:1080" (master listening with SOCKS proxy)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
//
// The test creates:
// 1. A mock destination server at 127.0.0.1:8080 on the slave side
// 2. A SOCKS5 client connecting to the proxy at 127.0.0.1:1080 on the master side
// 3. Verifies that data flows correctly through the SOCKS proxy tunnel
func TestSocksConnect(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Setup mock destination server on slave side (this would be at 127.0.0.1:8080)
	// This server will respond with unique data when contacted via SOCKS proxy
	// Use the standard echo server pattern for consistency
	srv, err := mocks_tcp.NewServer(setup.TCPNetwork.ListenTCP, "tcp", "127.0.0.1:8080", "DESTINATION_SERVER_RESPONSE: ")
	if err != nil {
		t.Fatalf("Failed to start destination server: %v", err)
	}
	defer srv.Close()

	// Configure master with SOCKS proxy
	// Simulates "master listen 'tcp://*:12345' -D 127.0.0.1:1080"
	setup.MasterCfg.Socks = config.NewSocksCfg("127.0.0.1:1080")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server using entrypoint (listens for connections and sets up SOCKS proxy)
	go func() {
		if err := entrypoint.MasterListen(ctx, setup.MasterSharedCfg, setup.MasterCfg); err != nil {
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

	// Wait for slave to connect to master
	var cSlaveToMaster *mocks_tcp.MockTCPConn
	if cSlaveToMaster, err = lMaster.WaitForNewConnection(2000); err != nil {
		t.Fatalf("Slave failed to connect to master: %v", err)
	}
	t.Logf("Slave is connected to master: %s->%s", cSlaveToMaster.RemoteAddr().String(), cSlaveToMaster.LocalAddr().String())

	// Wait for the SOCKS proxy to be available
	var lSocks *mocks_tcp.MockTCPListener
	if lSocks, err = setup.TCPNetwork.WaitForListener("127.0.0.1:1080", 2000); err != nil {
		t.Fatalf("SOCKS proxy failed to start listening: %v", err)
	}
	t.Logf("SOCKS proxy has started listening on %s", lSocks.Addr().String())

	// Wait for destination server to be available
	var lDest *mocks_tcp.MockTCPListener
	if lDest, err = setup.TCPNetwork.WaitForListener("127.0.0.1:8080", 2000); err != nil {
		t.Fatalf("Destination server failed to start listening: %v", err)
	}
	t.Logf("Destination server has started listening on %s", lDest.Addr().String())

	// Now test the SOCKS proxy by acting as a SOCKS5 client
	// Connect to the SOCKS proxy at 127.0.0.1:1080
	socksProxyAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:1080")
	if err != nil {
		t.Fatalf("Failed to resolve SOCKS proxy address: %v", err)
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

		// Connect to the SOCKS proxy
		socksConn, err := setup.TCPNetwork.DialTCP("tcp", nil, socksProxyAddr)
		if err != nil {
			clientErr = fmt.Errorf("failed to connect to SOCKS proxy: %v", err)
			return
		}
		defer socksConn.Close()

		// Perform SOCKS5 handshake
		// Step 1: Method selection request (no authentication)
		bufSocksConn := bufio.NewReadWriter(bufio.NewReader(socksConn), bufio.NewWriter(socksConn))

		// Send: VER (0x05), NMETHODS (1), METHOD (0x00 = no auth)
		methodRequest := []byte{socks.VersionSocks5, 0x01, byte(socks.MethodNoAuthenticationRequired)}
		if _, err := bufSocksConn.Write(methodRequest); err != nil {
			clientErr = fmt.Errorf("failed to send method selection: %v", err)
			return
		}
		if err := bufSocksConn.Flush(); err != nil {
			clientErr = fmt.Errorf("failed to flush method selection: %v", err)
			return
		}

		// Receive method selection response
		methodResponse := make([]byte, 2)
		if _, err := io.ReadFull(bufSocksConn, methodResponse); err != nil {
			clientErr = fmt.Errorf("failed to read method selection response: %v", err)
			return
		}
		if methodResponse[0] != socks.VersionSocks5 || methodResponse[1] != byte(socks.MethodNoAuthenticationRequired) {
			clientErr = fmt.Errorf("unexpected method selection response: %v", methodResponse)
			return
		}

		// Step 2: CONNECT request to 127.0.0.1:8080
		// Format: VER CMD RSV ATYP DST.ADDR DST.PORT
		connectRequest := []byte{
			socks.VersionSocks5,         // VER
			byte(socks.CommandConnect),  // CMD
			socks.RSV,                   // RSV
			byte(socks.AddressTypeIPv4), // ATYP (IPv4)
			127, 0, 0, 1,                // DST.ADDR (127.0.0.1)
			0x1F, 0x90, // DST.PORT (8080 in network byte order)
		}
		if _, err := bufSocksConn.Write(connectRequest); err != nil {
			clientErr = fmt.Errorf("failed to send CONNECT request: %v", err)
			return
		}
		if err := bufSocksConn.Flush(); err != nil {
			clientErr = fmt.Errorf("failed to flush CONNECT request: %v", err)
			return
		}

		// Receive CONNECT response
		// Format: VER REP RSV ATYP BND.ADDR BND.PORT
		connectResponse := make([]byte, 4)
		if _, err := io.ReadFull(bufSocksConn, connectResponse); err != nil {
			clientErr = fmt.Errorf("failed to read CONNECT response header: %v", err)
			return
		}
		if connectResponse[0] != socks.VersionSocks5 {
			clientErr = fmt.Errorf("unexpected SOCKS version in response: %d", connectResponse[0])
			return
		}
		if connectResponse[1] != byte(socks.ReplySuccess) {
			clientErr = fmt.Errorf("CONNECT request failed with reply code: %d", connectResponse[1])
			return
		}

		// Read the rest of the response based on address type
		atyp := connectResponse[3]
		var addrLen int
		switch socks.Atyp(atyp) {
		case socks.AddressTypeIPv4:
			addrLen = 4
		case socks.AddressTypeIPv6:
			addrLen = 16
		case socks.AddressTypeFQDN:
			// Read length byte first
			lenByte := make([]byte, 1)
			if _, err := io.ReadFull(bufSocksConn, lenByte); err != nil {
				clientErr = fmt.Errorf("failed to read FQDN length: %v", err)
				return
			}
			addrLen = int(lenByte[0])
		default:
			clientErr = fmt.Errorf("unexpected address type: %d", atyp)
			return
		}

		// Read BND.ADDR and BND.PORT
		remaining := make([]byte, addrLen+2) // addr + 2 bytes for port
		if _, err := io.ReadFull(bufSocksConn, remaining); err != nil {
			clientErr = fmt.Errorf("failed to read CONNECT response remaining: %v", err)
			return
		}

		// Step 3: Now we're connected through the SOCKS proxy to the destination server
		// Send test data through the tunnel
		testData := "Hello through SOCKS proxy!\n"
		if _, err := bufSocksConn.WriteString(testData); err != nil {
			clientErr = fmt.Errorf("failed to write test data: %v", err)
			return
		}
		if err := bufSocksConn.Flush(); err != nil {
			clientErr = fmt.Errorf("failed to flush test data: %v", err)
			return
		}

		// Read response from the destination server (should come through the SOCKS proxy)
		// Required, the piping through the relay is not immediate
		time.Sleep(100 * time.Millisecond)

		socksConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		line, err := bufSocksConn.ReadString('\n')
		if err != nil && err != io.EOF {
			clientErr = fmt.Errorf("failed to read response: %v", err)
			return
		}

		clientResponse = strings.TrimSpace(line)
		clientSuccess = true
	}()

	// Wait for client to complete
	clientWg.Wait()

	// Wait for the SOCKS client connection to arrive at the proxy
	var cClientToSocks *mocks_tcp.MockTCPConn
	if cClientToSocks, err = lSocks.WaitForNewConnection(2000); err != nil {
		t.Fatalf("SOCKS client failed to connect to proxy: %v", err)
	}
	t.Logf("SOCKS client connected to proxy: %s->%s", cClientToSocks.RemoteAddr().String(), cClientToSocks.LocalAddr().String())

	// Wait for the relay connection from SOCKS proxy to destination server
	var cRelayToDest *mocks_tcp.MockTCPConn
	if cRelayToDest, err = lDest.WaitForNewConnection(2000); err != nil {
		t.Fatalf("SOCKS relay failed to connect to destination server: %v", err)
	}
	t.Logf("SOCKS relay connected to destination server: %s->%s", cRelayToDest.RemoteAddr().String(), cRelayToDest.LocalAddr().String())

	// Verify the client test results
	if clientErr != nil {
		t.Errorf("Client error: %v", clientErr)
	}

	if !clientSuccess {
		t.Fatal("Client failed to complete successfully")
	}

	// Verify the response contains both the expected prefix and the sent data
	expectedPrefix := "DESTINATION_SERVER_RESPONSE:"
	expectedContent := "Hello through SOCKS proxy!"
	if !strings.Contains(clientResponse, expectedPrefix) {
		t.Errorf("Expected response to contain '%s', got: %q", expectedPrefix, clientResponse)
	}
	if !strings.Contains(clientResponse, expectedContent) {
		t.Errorf("Expected response to contain sent data '%s', got: %q", expectedContent, clientResponse)
	}

	t.Logf("✓ SOCKS proxy test successful! Response: %q", clientResponse)

	// Test multiple connections to ensure SOCKS proxy is stable
	for i := 0; i < 3; i++ {
		clientWg.Add(1)
		go func(iteration int) {
			defer clientWg.Done()

			socksConn, err := setup.TCPNetwork.DialTCP("tcp", nil, socksProxyAddr)
			if err != nil {
				t.Errorf("Iteration %d: failed to connect: %v", iteration, err)
				return
			}
			defer socksConn.Close()

			bufSocksConn := bufio.NewReadWriter(bufio.NewReader(socksConn), bufio.NewWriter(socksConn))

			// Method selection
			methodRequest := []byte{socks.VersionSocks5, 0x01, byte(socks.MethodNoAuthenticationRequired)}
			bufSocksConn.Write(methodRequest)
			bufSocksConn.Flush()

			methodResponse := make([]byte, 2)
			io.ReadFull(bufSocksConn, methodResponse)

			// CONNECT request
			connectRequest := []byte{
				socks.VersionSocks5, byte(socks.CommandConnect), socks.RSV,
				byte(socks.AddressTypeIPv4), 127, 0, 0, 1, 0x1F, 0x90,
			}
			bufSocksConn.Write(connectRequest)
			bufSocksConn.Flush()

			// Read CONNECT response
			connectResponse := make([]byte, 4)
			io.ReadFull(bufSocksConn, connectResponse)
			atyp := connectResponse[3]
			var addrLen int
			switch socks.Atyp(atyp) {
			case socks.AddressTypeIPv4:
				addrLen = 4
			case socks.AddressTypeIPv6:
				addrLen = 16
			}
			remaining := make([]byte, addrLen+2)
			io.ReadFull(bufSocksConn, remaining)

			// Send test data (add newline for line-oriented echo server)
			testData := fmt.Sprintf("Message %d\n", iteration)
			bufSocksConn.WriteString(testData)
			bufSocksConn.Flush()

			// Read response (required, the piping through the relay is not immediate)
			time.Sleep(100 * time.Millisecond)

			socksConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			line, err := bufSocksConn.ReadString('\n')
			if err != nil && err != io.EOF {
				t.Errorf("Iteration %d: failed to read: %v", iteration, err)
				return
			}

			response := strings.TrimSpace(line)
			expectedData := fmt.Sprintf("Message %d", iteration)
			if !strings.Contains(response, expectedData) {
				t.Errorf("Iteration %d: expected response to contain '%s', got: %q", iteration, expectedData, response)
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
