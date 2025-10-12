package associate

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestMultipleClients tests SOCKS5 UDP ASSOCIATE with multiple concurrent clients.
// Each client should get its own UDP relay and be able to send/receive independently.
func TestMultipleClients(t *testing.T) {
	setup := SetupTest(t)
	defer setup.Cleanup()

	// Number of concurrent clients to test
	numClients := 3

	type clientResult struct {
		clientID int
		success  bool
		err      error
		response string
	}

	var wg sync.WaitGroup
	results := make(chan clientResult, numClients)

	// Create multiple clients concurrently
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			t.Logf("[Client %d] Starting client goroutine", clientID)

			// Create SOCKS client
			client := CreateSOCKSClient(t, setup)
			if client == nil {
				results <- clientResult{clientID: clientID, success: false, err: fmt.Errorf("failed to create client")}
				return
			}
			defer client.Close()

			t.Logf("[Client %d] Created SOCKS client, UDP relay at %s", clientID, client.UDPRelayAddr)

			// Send UDP datagram with unique data
			testData := fmt.Sprintf("Message from client %d", clientID)
			t.Logf("[Client %d] Sending UDP datagram: %q", clientID, testData)

			response, err := client.SendUDPDatagram(t, testData)
			if err != nil {
				t.Logf("[Client %d] Error: %v", clientID, err)
				results <- clientResult{clientID: clientID, success: false, err: err}
				return
			}

			t.Logf("[Client %d] Received response: %q", clientID, response)

			// Verify the response contains the unique data
			expectedPrefix := "UDP_SERVER_RESPONSE:"
			if !strings.Contains(response, expectedPrefix) {
				results <- clientResult{
					clientID: clientID,
					success:  false,
					err:      fmt.Errorf("response missing expected prefix '%s'", expectedPrefix),
					response: response,
				}
				return
			}

			if !strings.Contains(response, testData) {
				results <- clientResult{
					clientID: clientID,
					success:  false,
					err:      fmt.Errorf("response missing sent data '%s'", testData),
					response: response,
				}
				return
			}

			results <- clientResult{clientID: clientID, success: true, response: response}
			t.Logf("[Client %d] ✓ Test successful!", clientID)
		}(i)
	}

	// Wait for all clients to complete
	wg.Wait()
	close(results)

	// Check results
	successCount := 0
	for result := range results {
		if result.success {
			successCount++
			t.Logf("✓ Client %d successful! Response: %q", result.clientID, result.response)
		} else {
			t.Errorf("✗ Client %d failed: %v (response: %q)", result.clientID, result.err, result.response)
		}
	}

	if successCount != numClients {
		t.Errorf("Only %d out of %d clients succeeded", successCount, numClients)
	} else {
		t.Logf("✓ All %d clients completed successfully!", numClients)
	}

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
