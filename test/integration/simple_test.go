package integration

import (
	"context"
	"dominicbreuker/goncat/mocks"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"io"
	"strings"
	"testing"
	"time"
)

// TestEndToEndDataExchange simulates a complete master-slave connection
// with mocked network and stdio, demonstrating full end-to-end data flow.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345'" (master listening)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
func TestEndToEndDataExchange(t *testing.T) {
	// Create mock network for TCP connections
	mockNet := mocks.NewMockTCPNetwork()

	// Create mock stdio for master and slave
	masterStdio := mocks.NewMockStdio()
	slaveStdio := mocks.NewMockStdio()
	defer masterStdio.Close()
	defer slaveStdio.Close()

	// Setup master dependencies (network + stdio)
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

	// Master configuration - simulates "master listen 'tcp://*:12345'"
	masterSharedCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Deps:     masterDeps,
	}

	masterCfg := &config.Master{
		Exec:    "", // No exec, just foreground piping
		Pty:     false,
		LogFile: "",
	}

	// Slave configuration - simulates "slave connect tcp://127.0.0.1:12345"
	slaveSharedCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Deps:     slaveDeps,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Channels for synchronization and error handling
	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master server using entrypoint (listens for connections)
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

	// Give master time to start listening
	time.Sleep(200 * time.Millisecond)

	// Start slave using entrypoint (connects to master)
	go func() {
		if err := entrypoint.SlaveConnect(ctx, slaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	// Give connection time to establish and handlers to start
	time.Sleep(300 * time.Millisecond)

	// Test master → slave data flow
	masterInput := "Hello from master!\n"
	masterStdio.WriteToStdin([]byte(masterInput))

	// Wait for data to flow through the network
	time.Sleep(500 * time.Millisecond)

	// Verify data arrived at slave's stdout
	slaveOutput := slaveStdio.ReadFromStdout()
	if !strings.Contains(slaveOutput, "Hello from master!") {
		t.Errorf("Expected slave stdout to contain 'Hello from master!', got: %q", slaveOutput)
	}

	// Test slave → master data flow (bidirectional)
	slaveInput := "Hello from slave!\n"
	slaveStdio.WriteToStdin([]byte(slaveInput))

	// Wait for data to flow back through the network
	time.Sleep(500 * time.Millisecond)

	// Verify data arrived at master's stdout
	masterOutput := masterStdio.ReadFromStdout()
	if !strings.Contains(masterOutput, "Hello from slave!") {
		t.Errorf("Expected master stdout to contain 'Hello from slave!', got: %q", masterOutput)
	}

	// Test multiple messages to ensure continuous bidirectional communication
	masterInput2 := "Second message from master\n"
	masterStdio.WriteToStdin([]byte(masterInput2))
	time.Sleep(300 * time.Millisecond)

	slaveOutput2 := slaveStdio.ReadFromStdout()
	if !strings.Contains(slaveOutput2, "Second message from master") {
		t.Errorf("Expected slave to receive second message, got: %q", slaveOutput2)
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
