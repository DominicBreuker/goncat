package associate

import (
	"strings"
	"testing"
)

// TestSingleClient tests SOCKS5 UDP ASSOCIATE with a single client.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345' -D 127.0.0.1:1080" (master listening with SOCKS proxy)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
//
// The test validates complete packet flow:
// 1. Client sends UDP datagram to master relay
// 2. Master parses SOCKS5 header and forwards to slave via TCP
// 3. Slave sends UDP to destination server
// 4. Destination responds
// 5. Slave captures response with source address and forwards to master via TCP
// 6. Master sends UDP response back to client with proper SOCKS5 header
// 7. Client receives and validates the response
func TestSingleClient(t *testing.T) {
	setup := SetupTest(t)
	defer setup.Cleanup()

	// Create SOCKS client
	client := CreateSOCKSClient(t, setup)
	defer client.Close()

	// Send UDP datagram
	testData := "Hello via UDP SOCKS proxy!"
	response, err := client.SendUDPDatagram(t, testData)
	if err != nil {
		t.Fatalf("Failed to send/receive UDP datagram: %v", err)
	}

	// Verify the response
	expectedPrefix := "UDP_SERVER_RESPONSE:"
	if !strings.Contains(response, expectedPrefix) {
		t.Errorf("Expected response to contain '%s', got: %q", expectedPrefix, response)
	}
	if !strings.Contains(response, testData) {
		t.Errorf("Expected response to contain sent data '%s', got: %q", testData, response)
	}

	t.Logf("✓ SOCKS UDP proxy test successful! Response: %q", response)

	// Cleanup
	setup.Cancel()

	// Check for errors (non-blocking)
	select {
	case err := <-setup.MasterErr:
		if err != nil {
			t.Logf("Master completed with: %v", err)
		}
	default:
		t.Log("Master still running (expected)")
	}

	select {
	case err := <-setup.SlaveErr:
		if err != nil {
			t.Logf("Slave completed with: %v", err)
		}
	default:
		t.Log("Slave still running (expected)")
	}
}
