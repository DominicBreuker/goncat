package associate

import (
	"bufio"
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/pkg/socks"
	"dominicbreuker/goncat/test/helpers"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// TestSetup contains all the test infrastructure for SOCKS ASSOCIATE tests
type TestSetup struct {
	Setup          *helpers.MockDependenciesAndConfigs
	DestServerAddr *net.UDPAddr
	DestListener   net.PacketConn
	SocksProxyAddr *net.TCPAddr
	Ctx            context.Context
	Cancel         context.CancelFunc
	MasterErr      chan error
	SlaveErr       chan error
}

// SetupTest creates and initializes all test infrastructure
func SetupTest(t *testing.T) *TestSetup {
	t.Helper()

	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()

	// Setup mock UDP destination server on slave side (this would be at 127.0.0.1:8080)
	destServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to resolve destination server address: %v", err)
	}

	destServerListener, err := setup.UDPNetwork.ListenUDP("udp", destServerAddr)
	if err != nil {
		t.Fatalf("Failed to create destination server listener: %v", err)
	}

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

	// Configure master with SOCKS proxy
	// Simulates "master listen 'tcp://*:12345' -D 127.0.0.1:1080"
	setup.MasterCfg.Socks = config.NewSocksCfg("127.0.0.1:1080")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server using entrypoint
	go func() {
		if err := entrypoint.MasterListen(ctx, setup.MasterSharedCfg, setup.MasterCfg); err != nil {
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
	if _, err := setup.TCPNetwork.WaitForListener("127.0.0.1:12345", 2000); err != nil {
		t.Fatalf("Master failed to start listening: %v", err)
	}

	// Start slave using entrypoint
	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	// Wait for the SOCKS proxy to be available
	if _, err := setup.TCPNetwork.WaitForListener("127.0.0.1:1080", 2000); err != nil {
		t.Fatalf("SOCKS proxy failed to start listening: %v", err)
	}

	socksProxyAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:1080")

	return &TestSetup{
		Setup:          setup,
		DestServerAddr: destServerAddr,
		DestListener:   destServerListener,
		SocksProxyAddr: socksProxyAddr,
		Ctx:            ctx,
		Cancel:         cancel,
		MasterErr:      masterErr,
		SlaveErr:       slaveErr,
	}
}

// Cleanup closes all resources
func (s *TestSetup) Cleanup() {
	s.Setup.Close()
	s.DestListener.Close()
	s.Cancel()
}

// SOCKSClient represents a SOCKS5 UDP client
type SOCKSClient struct {
	TCPConn      net.Conn
	UDPConn      net.PacketConn
	UDPRelayAddr *net.UDPAddr
}

// CreateSOCKSClient creates a new SOCKS5 client and performs the handshake
func CreateSOCKSClient(t *testing.T, setup *TestSetup) *SOCKSClient {
	t.Helper()

	// Connect to SOCKS proxy
	socksConn, err := setup.Setup.TCPNetwork.DialTCP("tcp", nil, setup.SocksProxyAddr)
	if err != nil {
		t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
		return nil
	}

	bufSocksConn := bufio.NewReadWriter(bufio.NewReader(socksConn), bufio.NewWriter(socksConn))

	// Perform SOCKS5 handshake - Method selection
	methodRequest := []byte{socks.VersionSocks5, 0x01, byte(socks.MethodNoAuthenticationRequired)}
	if _, err := bufSocksConn.Write(methodRequest); err != nil {
		t.Fatalf("Failed to send method selection: %v", err)
		return nil
	}
	if err := bufSocksConn.Flush(); err != nil {
		t.Fatalf("Failed to flush method selection: %v", err)
		return nil
	}

	// Receive method selection response
	methodResponse := make([]byte, 2)
	if _, err := io.ReadFull(bufSocksConn, methodResponse); err != nil {
		t.Fatalf("Failed to read method selection response: %v", err)
		return nil
	}

	// Send UDP ASSOCIATE request
	associateRequest := []byte{
		socks.VersionSocks5,          // VER
		byte(socks.CommandAssociate), // CMD
		socks.RSV,                    // RSV
		byte(socks.AddressTypeIPv4),  // ATYP
		0, 0, 0, 0,                   // DST.ADDR (0.0.0.0)
		0, 0, // DST.PORT (0)
	}
	if _, err := bufSocksConn.Write(associateRequest); err != nil {
		t.Fatalf("Failed to send ASSOCIATE request: %v", err)
		return nil
	}
	if err := bufSocksConn.Flush(); err != nil {
		t.Fatalf("Failed to flush ASSOCIATE request: %v", err)
		return nil
	}

	// Receive ASSOCIATE response
	associateResponse := make([]byte, 4)
	if _, err := io.ReadFull(bufSocksConn, associateResponse); err != nil {
		t.Fatalf("Failed to read ASSOCIATE response header: %v", err)
		return nil
	}

	// Read the rest of the response to get UDP relay address
	atyp := associateResponse[3]
	var addrLen int
	switch socks.Atyp(atyp) {
	case socks.AddressTypeIPv4:
		addrLen = 4
	case socks.AddressTypeIPv6:
		addrLen = 16
	case socks.AddressTypeFQDN:
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(bufSocksConn, lenByte); err != nil {
			t.Fatalf("Failed to read FQDN length: %v", err)
			return nil
		}
		addrLen = int(lenByte[0])
	}

	remaining := make([]byte, addrLen+2)
	if _, err := io.ReadFull(bufSocksConn, remaining); err != nil {
		t.Fatalf("Failed to read ASSOCIATE response remaining: %v", err)
		return nil
	}

	// Extract UDP relay address
	var udpRelayIP net.IP
	if atyp == byte(socks.AddressTypeIPv4) {
		udpRelayIP = net.IPv4(remaining[0], remaining[1], remaining[2], remaining[3])
	} else if atyp == byte(socks.AddressTypeIPv6) {
		udpRelayIP = remaining[:16]
	}
	udpRelayPort := int(remaining[addrLen])<<8 | int(remaining[addrLen+1])
	udpRelayAddr := &net.UDPAddr{IP: udpRelayIP, Port: udpRelayPort}

	// Wait for the UDP relay to be ready
	if err := setup.Setup.UDPNetwork.WaitForListener(udpRelayAddr.String(), 2000); err != nil {
		t.Fatalf("UDP relay failed to start: %v", err)
		return nil
	}

	// Create UDP connection for the client
	clientUDPAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	clientUDPConn, err := setup.Setup.UDPNetwork.ListenUDP("udp", clientUDPAddr)
	if err != nil {
		t.Fatalf("Failed to create client UDP connection: %v", err)
		return nil
	}

	return &SOCKSClient{
		TCPConn:      socksConn,
		UDPConn:      clientUDPConn,
		UDPRelayAddr: udpRelayAddr,
	}
}

// SendUDPDatagram sends a SOCKS5 UDP datagram and waits for response
func (c *SOCKSClient) SendUDPDatagram(t *testing.T, data string) (string, error) {
	t.Helper()

	// Build SOCKS5 UDP datagram
	udpDatagram := []byte{
		socks.RSV, socks.RSV, socks.FRAG, // RSV, RSV, FRAG=0
		byte(socks.AddressTypeIPv4), // ATYP
		127, 0, 0, 1,                // DST.ADDR (127.0.0.1)
		0x1F, 0x90, // DST.PORT (8080)
	}
	udpDatagram = append(udpDatagram, []byte(data)...)

	// Send datagram
	_, err := c.UDPConn.WriteTo(udpDatagram, c.UDPRelayAddr)
	if err != nil {
		return "", fmt.Errorf("failed to send UDP datagram: %v", err)
	}

	// Wait for response (using read deadline for timeout)
	responseBuf := make([]byte, 65507)
	c.UDPConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := c.UDPConn.ReadFrom(responseBuf)
	if err != nil {
		return "", fmt.Errorf("failed to read UDP response: %v", err)
	}

	// Parse SOCKS5 UDP header (skip 10 bytes for IPv4)
	if n < 10 {
		return "", fmt.Errorf("response too short: %d bytes", n)
	}

	responseData := string(responseBuf[10:n])
	return responseData, nil
}

// Close closes the client connections
func (c *SOCKSClient) Close() {
	if c.TCPConn != nil {
		c.TCPConn.Close()
	}
	if c.UDPConn != nil {
		c.UDPConn.Close()
	}
}
