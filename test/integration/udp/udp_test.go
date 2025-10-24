package udp

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/test/helpers"
	"testing"
	"time"
)

// TestUDPEndToEndDataExchange tests UDP transport with master listen / slave connect.
// This test mimics the behavior of:
//   - "goncat master listen 'udp://*:12345'" (master listening)
//   - "goncat slave connect udp://127.0.0.1:12345" (slave connecting)
func TestUDPEndToEndDataExchange(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure for UDP protocol
	setup.MasterSharedCfg.Protocol = config.ProtoUDP
	setup.SlaveSharedCfg.Protocol = config.ProtoUDP

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
	// Note: UDP mock doesn't need WaitForListener like TCP, but we add a delay
	time.Sleep(200 * time.Millisecond)

	// Start slave using entrypoint (connects to master)
	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	// Wait a moment for connection to establish
	time.Sleep(300 * time.Millisecond)

	// Test master → slave data flow
	masterInput := "Hello via UDP!\n"
	setup.MasterStdio.WriteToStdin([]byte(masterInput))

	// Wait for data to arrive at slave's stdout
	if err := setup.SlaveStdio.WaitForOutput("Hello via UDP!", 2000); err != nil {
		t.Errorf("UDP data did not arrive at slave: %v", err)
	}

	// Test slave → master data flow (bidirectional)
	slaveInput := "Response via UDP!\n"
	setup.SlaveStdio.WriteToStdin([]byte(slaveInput))

	// Wait for data to arrive at master's stdout
	if err := setup.MasterStdio.WaitForOutput("Response via UDP!", 2000); err != nil {
		t.Errorf("UDP data did not arrive at master: %v", err)
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

// TestUDPWithAuthentication tests UDP transport with --ssl and --key flags.
func TestUDPWithAuthentication(t *testing.T) {
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure for UDP protocol with authentication
	setup.MasterSharedCfg.Protocol = config.ProtoUDP
	setup.MasterSharedCfg.SSL = true
	setup.MasterSharedCfg.Key = "testsecret"

	setup.SlaveSharedCfg.Protocol = config.ProtoUDP
	setup.SlaveSharedCfg.SSL = true
	setup.SlaveSharedCfg.Key = "testsecret"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	masterErr := make(chan error, 1)
	slaveErr := make(chan error, 1)

	// Start master
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

	time.Sleep(200 * time.Millisecond)

	// Start slave
	go func() {
		if err := entrypoint.SlaveConnect(ctx, setup.SlaveSharedCfg); err != nil {
			slaveErr <- err
			return
		}
		slaveErr <- nil
	}()

	time.Sleep(300 * time.Millisecond)

	// Test authenticated data exchange
	setup.MasterStdio.WriteToStdin([]byte("Authenticated message\n"))

	if err := setup.SlaveStdio.WaitForOutput("Authenticated message", 2000); err != nil {
		t.Errorf("Authenticated UDP data did not arrive: %v", err)
	}

	cancel()
	time.Sleep(200 * time.Millisecond)
}

// TestUDPMasterConnect tests the reverse topology: slave listen, master connect.
func TestUDPMasterConnect(t *testing.T) {
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure for UDP protocol
	setup.MasterSharedCfg.Protocol = config.ProtoUDP
	setup.SlaveSharedCfg.Protocol = config.ProtoUDP

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slaveErr := make(chan error, 1)
	masterErr := make(chan error, 1)

	// Start slave listener first
	go func() {
		if err := entrypoint.SlaveListen(ctx, setup.SlaveSharedCfg); err != nil {
			select {
			case <-ctx.Done():
				slaveErr <- nil
			default:
				slaveErr <- err
			}
			return
		}
		slaveErr <- nil
	}()

	time.Sleep(200 * time.Millisecond)

	// Master connects to slave
	go func() {
		if err := entrypoint.MasterConnect(ctx, setup.MasterSharedCfg, setup.MasterCfg); err != nil {
			masterErr <- err
			return
		}
		masterErr <- nil
	}()

	time.Sleep(300 * time.Millisecond)

	// Test data exchange in reverse topology
	setup.MasterStdio.WriteToStdin([]byte("Master to slave\n"))

	if err := setup.SlaveStdio.WaitForOutput("Master to slave", 2000); err != nil {
		t.Errorf("Data did not arrive in reverse topology: %v", err)
	}

	cancel()
	time.Sleep(200 * time.Millisecond)
}
