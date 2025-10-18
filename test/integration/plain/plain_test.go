package plain

import (
	"context"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/test/helpers"
	"testing"
	"time"
)

// TestEndToEndDataExchange simulates a complete master-slave connection
// with mocked network and stdio, demonstrating full end-to-end data flow.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345'" (master listening)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
func TestEndToEndDataExchange(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// No additional configuration needed for this simple test
	// (using defaults: TCP protocol, 127.0.0.1:12345, no exec, just foreground piping)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server using entrypoint (listens for connections)
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
	if _, err := setup.TCPNetwork.WaitForListener("127.0.0.1:12345", 2000); err != nil {
		t.Fatalf("Master failed to start listening: %v", err)
	}

	// Start slave using entrypoint (connects to master)
	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	// Test master → slave data flow
	masterInput := "Hello from master!\n"
	setup.MasterStdio.WriteToStdin([]byte(masterInput))

	// Wait for data to arrive at slave's stdout
	if err := setup.SlaveStdio.WaitForOutput("Hello from master!", 2000); err != nil {
		t.Errorf("Data did not arrive at slave: %v", err)
	}

	// Test slave → master data flow (bidirectional)
	slaveInput := "Hello from slave!\n"
	setup.SlaveStdio.WriteToStdin([]byte(slaveInput))

	// Wait for data to arrive at master's stdout
	if err := setup.MasterStdio.WaitForOutput("Hello from slave!", 2000); err != nil {
		t.Errorf("Data did not arrive at master: %v", err)
	}

	// Test multiple messages to ensure continuous bidirectional communication
	masterInput2 := "Second message from master\n"
	setup.MasterStdio.WriteToStdin([]byte(masterInput2))

	// Wait for second message to arrive at slave's stdout
	if err := setup.SlaveStdio.WaitForOutput("Second message from master", 2000); err != nil {
		t.Errorf("Second message did not arrive at slave: %v", err)
	}

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
