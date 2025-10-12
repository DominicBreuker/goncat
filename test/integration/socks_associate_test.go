package integration

import (
	"bufio"
	"context"
	"dominicbreuker/goncat/mocks"
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

// TestSocksAssociate simulates a complete SOCKS5 UDP proxy scenario with mocked network.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345' -D 127.0.0.1:1080" (master listening with SOCKS proxy)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
//
// The test creates:
// 1. A mock UDP destination server at 127.0.0.1:8080 on the slave side
// 2. A SOCKS5 client connecting to the proxy at 127.0.0.1:1080 on the master side
// 3. Verifies that UDP datagrams flow correctly through the SOCKS proxy tunnel
//
// TODO: This test currently hangs. The infrastructure (UDP mocking, dependency injection) is in place,
// but there appears to be a synchronization issue where UDP packets aren't flowing through the relay chain.
// The relay goroutines may not be starting properly, or there may be a deadlock in the UDP packet flow.
// Further debugging needed to identify the root cause.
func TestSocksAssociate(t *testing.T) {
	// Create mock networks for TCP and UDP connections
	mockTCPNet := mocks.NewMockTCPNetwork()
	mockUDPNet := mocks.NewMockUDPNetwork()

	// Create mock stdio for master and slave (not used in this test but required for setup)
	masterStdio := mocks.NewMockStdio()
	slaveStdio := mocks.NewMockStdio()
	defer masterStdio.Close()
	defer slaveStdio.Close()

	// Setup mock UDP destination server on slave side (this would be at 127.0.0.1:8080)
	// This server will respond with unique data when contacted via SOCKS proxy
	destServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to resolve destination server address: %v", err)
	}

	destServerListener, err := mockUDPNet.ListenUDP("udp", destServerAddr)
	if err != nil {
		t.Fatalf("Failed to create destination server listener: %v", err)
	}
	defer destServerListener.Close()

	// Start mock UDP destination server in a goroutine
	destServerStarted := make(chan struct{})
	go func() {
		close(destServerStarted)
		buf := make([]byte, 65507)
		for {
			n, clientAddr, err := destServerListener.ReadFrom(buf)
			if err != nil {
				return // listener closed
			}
			// Read request data and respond
			request := string(buf[:n])
			response := fmt.Sprintf("UDP_SERVER_RESPONSE: You sent '%s'", strings.TrimSpace(request))
			destServerListener.WriteTo([]byte(response), clientAddr)
		}
	}()

	// Wait for destination server to start
	<-destServerStarted

	// Setup master dependencies (network + stdio)
	masterDeps := &config.Dependencies{
		TCPDialer:      mockTCPNet.DialTCP,
		TCPListener:    mockTCPNet.ListenTCP,
		UDPListener:    mockUDPNet.ListenUDP,
		PacketListener: mockUDPNet.ListenPacket,
		Stdin:          func() io.Reader { return masterStdio.GetStdin() },
		Stdout:         func() io.Writer { return masterStdio.GetStdout() },
	}

	// Setup slave dependencies (network + stdio)
	slaveDeps := &config.Dependencies{
		TCPDialer:      mockTCPNet.DialTCP,
		TCPListener:    mockTCPNet.ListenTCP,
		UDPListener:    mockUDPNet.ListenUDP,
		PacketListener: mockUDPNet.ListenPacket,
		Stdin:          func() io.Reader { return slaveStdio.GetStdin() },
		Stdout:         func() io.Writer { return slaveStdio.GetStdout() },
	}

	// Master configuration with SOCKS proxy
	// Simulates "master listen 'tcp://*:12345' -D 127.0.0.1:1080"
	masterSharedCfg := helpers.DefaultSharedConfig(masterDeps)
	masterCfg := helpers.DefaultMasterConfig()
	masterCfg.Socks = config.NewSocksCfg("127.0.0.1:1080")

	// Slave configuration - simulates "slave connect tcp://127.0.0.1:12345"
	slaveSharedCfg := helpers.DefaultSharedConfig(slaveDeps)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server using entrypoint (listens for connections and sets up SOCKS proxy)
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
	if err := mockTCPNet.WaitForListener("127.0.0.1:12345", 2000); err != nil {
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

	// Wait for the SOCKS proxy to be available
	if err := mockTCPNet.WaitForListener("127.0.0.1:1080", 2000); err != nil {
		t.Fatalf("SOCKS proxy failed to start listening: %v", err)
	}

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

		// Connect to the SOCKS proxy via TCP for control
		socksConn, err := mockTCPNet.DialTCP("tcp", nil, socksProxyAddr)
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

		// Step 2: UDP ASSOCIATE request to establish UDP relay
		// Format: VER CMD RSV ATYP DST.ADDR DST.PORT
		// For ASSOCIATE, DST.ADDR and DST.PORT can be zero if unknown
		associateRequest := []byte{
			socks.VersionSocks5,          // VER
			byte(socks.CommandAssociate), // CMD
			socks.RSV,                    // RSV
			byte(socks.AddressTypeIPv4),  // ATYP (IPv4)
			0, 0, 0, 0,                   // DST.ADDR (0.0.0.0 - not known yet)
			0, 0, // DST.PORT (0 - not known yet)
		}
		if _, err := bufSocksConn.Write(associateRequest); err != nil {
			clientErr = fmt.Errorf("failed to send ASSOCIATE request: %v", err)
			return
		}
		if err := bufSocksConn.Flush(); err != nil {
			clientErr = fmt.Errorf("failed to flush ASSOCIATE request: %v", err)
			return
		}

		// Receive ASSOCIATE response
		// Format: VER REP RSV ATYP BND.ADDR BND.PORT
		// BND.ADDR and BND.PORT indicate where to send UDP datagrams
		associateResponse := make([]byte, 4)
		if _, err := io.ReadFull(bufSocksConn, associateResponse); err != nil {
			clientErr = fmt.Errorf("failed to read ASSOCIATE response header: %v", err)
			return
		}
		if associateResponse[0] != socks.VersionSocks5 {
			clientErr = fmt.Errorf("unexpected SOCKS version in response: %d", associateResponse[0])
			return
		}
		if associateResponse[1] != byte(socks.ReplySuccess) {
			clientErr = fmt.Errorf("ASSOCIATE request failed with reply code: %d", associateResponse[1])
			return
		}

		// Read the rest of the response based on address type to get UDP relay address
		atyp := associateResponse[3]
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
			clientErr = fmt.Errorf("failed to read ASSOCIATE response remaining: %v", err)
			return
		}

		// Extract the UDP relay address and port from the response
		var udpRelayIP net.IP
		if atyp == byte(socks.AddressTypeIPv4) {
			udpRelayIP = net.IPv4(remaining[0], remaining[1], remaining[2], remaining[3])
		} else if atyp == byte(socks.AddressTypeIPv6) {
			udpRelayIP = remaining[:16]
		}
		udpRelayPort := int(remaining[addrLen])<<8 | int(remaining[addrLen+1])

		udpRelayAddr := &net.UDPAddr{
			IP:   udpRelayIP,
			Port: udpRelayPort,
		}

		// Give the relay goroutines time to start
		// This includes time for the slave to create its relay
		time.Sleep(500 * time.Millisecond)

		// Step 3: Now create a UDP connection to send datagrams through the proxy
		// The client needs to bind a local UDP port
		clientUDPAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		if err != nil {
			clientErr = fmt.Errorf("failed to resolve client UDP address: %v", err)
			return
		}

		clientUDPConn, err := mockUDPNet.ListenUDP("udp", clientUDPAddr)
		if err != nil {
			clientErr = fmt.Errorf("failed to create client UDP connection: %v", err)
			return
		}
		defer clientUDPConn.Close()

		// Build a SOCKS5 UDP datagram to send to the destination
		// Format: RSV RSV FRAG ATYP DST.ADDR DST.PORT DATA
		testData := "Hello via UDP SOCKS proxy!"
		udpDatagram := []byte{
			socks.RSV, socks.RSV, socks.FRAG, // RSV, RSV, FRAG=0
			byte(socks.AddressTypeIPv4), // ATYP
			127, 0, 0, 1,                // DST.ADDR (127.0.0.1 - destination server)
			0x1F, 0x90, // DST.PORT (8080 in network byte order)
		}
		udpDatagram = append(udpDatagram, []byte(testData)...)

		// Send the UDP datagram to the relay address
		_, err = clientUDPConn.WriteTo(udpDatagram, udpRelayAddr)
		if err != nil {
			clientErr = fmt.Errorf("failed to send UDP datagram: %v", err)
			return
		}

		// Give some time for the datagram to be processed through the entire chain
		time.Sleep(500 * time.Millisecond)

		// Wait for UDP response from the destination server (via the proxy)
		responseBuf := make([]byte, 65507)
		clientUDPConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := clientUDPConn.ReadFrom(responseBuf)
		if err != nil {
			clientErr = fmt.Errorf("failed to read UDP response: %v", err)
			return
		}

		// Parse the SOCKS5 UDP response header
		// Format: RSV RSV FRAG ATYP DST.ADDR DST.PORT DATA
		if n < 10 { // Minimum: 3 (RSV,RSV,FRAG) + 1 (ATYP) + 4 (IPv4) + 2 (port)
			clientErr = fmt.Errorf("UDP response too short: %d bytes", n)
			return
		}

		// Skip the SOCKS5 UDP header to get to the data
		// RSV(1) + RSV(1) + FRAG(1) + ATYP(1) + IPv4(4) + PORT(2) = 10 bytes
		responseData := responseBuf[10:n]
		clientResponse = string(responseData)
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
	expectedPrefix := "UDP_SERVER_RESPONSE:"
	expectedContent := "Hello via UDP SOCKS proxy!"
	if !strings.Contains(clientResponse, expectedPrefix) {
		t.Errorf("Expected response to contain '%s', got: %q", expectedPrefix, clientResponse)
	}
	if !strings.Contains(clientResponse, expectedContent) {
		t.Errorf("Expected response to contain sent data '%s', got: %q", expectedContent, clientResponse)
	}

	t.Logf("✓ SOCKS UDP proxy test successful! Response: %q", clientResponse)

	// TODO: Test multiple UDP datagrams to ensure SOCKS proxy is stable
	// Currently disabled because each iteration creates a new SOCKS connection
	// which creates a new UDP relay, and we need to properly handle multiple relays
	// or reuse the same UDP relay for multiple datagrams
	/*
		for i := 0; i < 3; i++ {
			clientWg.Add(1)
			go func(iteration int) {
				defer clientWg.Done()

				// Connect to SOCKS proxy
				socksConn, err := mockTCPNet.DialTCP("tcp", nil, socksProxyAddr)
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

				// ASSOCIATE request
				associateRequest := []byte{
					socks.VersionSocks5, byte(socks.CommandAssociate), socks.RSV,
					byte(socks.AddressTypeIPv4), 0, 0, 0, 0, 0, 0,
				}
				bufSocksConn.Write(associateRequest)
				bufSocksConn.Flush()

				// Read ASSOCIATE response
				associateResponse := make([]byte, 4)
				io.ReadFull(bufSocksConn, associateResponse)
				atyp := associateResponse[3]
				var addrLen int
				switch socks.Atyp(atyp) {
				case socks.AddressTypeIPv4:
					addrLen = 4
				case socks.AddressTypeIPv6:
					addrLen = 16
				}
				remaining := make([]byte, addrLen+2)
				io.ReadFull(bufSocksConn, remaining)

				// Extract UDP relay address
				var udpRelayIP net.IP
				if atyp == byte(socks.AddressTypeIPv4) {
					udpRelayIP = net.IPv4(remaining[0], remaining[1], remaining[2], remaining[3])
				}
				udpRelayPort := int(remaining[addrLen])<<8 | int(remaining[addrLen+1])
				udpRelayAddr := &net.UDPAddr{IP: udpRelayIP, Port: udpRelayPort}

				// Create client UDP connection
				clientUDPAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
				clientUDPConn, err := mockUDPNet.ListenUDP("udp", clientUDPAddr)
				if err != nil {
					t.Errorf("Iteration %d: failed to create UDP conn: %v", iteration, err)
					return
				}
				defer clientUDPConn.Close()

				// Send UDP datagram
				testData := fmt.Sprintf("UDP Message %d", iteration)
				udpDatagram := []byte{
					socks.RSV, socks.RSV, socks.FRAG,
					byte(socks.AddressTypeIPv4),
					127, 0, 0, 1,
					0x1F, 0x90,
				}
				udpDatagram = append(udpDatagram, []byte(testData)...)

				clientUDPConn.WriteTo(udpDatagram, udpRelayAddr)

				// Read response
				responseBuf := make([]byte, 65507)
				clientUDPConn.SetReadDeadline(time.Now().Add(2 * time.Second))
				n, _, err := clientUDPConn.ReadFrom(responseBuf)
				if err != nil {
					t.Errorf("Iteration %d: failed to read: %v", iteration, err)
					return
				}

				if n >= 10 {
					responseData := string(responseBuf[10:n])
					if !strings.Contains(responseData, testData) {
						t.Errorf("Iteration %d: expected response to contain '%s', got: %q", iteration, testData, responseData)
					} else {
						t.Logf("✓ Iteration %d successful! Response: %q", iteration, responseData)
					}
				}
			}(i)
		}

		// Wait for all iterations to complete
		clientWg.Wait()
	*/

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
