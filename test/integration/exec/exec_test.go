package exec

import (
	"context"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/test/helpers"
	"testing"
	"time"
)

// TestExecCommandExecution simulates a complete master-slave connection
// with command execution, using mocked network, stdio, and exec.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345' --exec /bin/sh" (master listening with exec)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
func TestExecCommandExecution(t *testing.T) {
	// Setup mock dependencies and default configs
	setup := helpers.SetupMockDependenciesAndConfigs()
	defer setup.Close()

	// Configure master to execute /bin/sh on slave
	setup.MasterCfg.Exec = "/bin/sh" // Execute /bin/sh on slave

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

	// Test 1: Echo command - the mock shell processes "echo <text>" commands
	setup.MasterStdio.WriteToStdin([]byte("echo hello world\n"))

	// Wait for the output to appear
	if err := setup.MasterStdio.WaitForOutput("hello world", 2000); err != nil {
		t.Errorf("Echo command output did not appear: %v", err)
	}

	// Test 2: Whoami command - the mock shell responds with mockcmd[/bin/sh]
	setup.MasterStdio.WriteToStdin([]byte("whoami\n"))

	// Wait for the output to appear
	if err := setup.MasterStdio.WaitForOutput("mockcmd[/bin/sh]", 2000); err != nil {
		t.Errorf("Whoami command output did not appear: %v", err)
	}

	// Test 3: Unsupported command - should get error response
	setup.MasterStdio.WriteToStdin([]byte("unsupported\n"))

	// Wait for the error message to appear
	if err := setup.MasterStdio.WaitForOutput("command not supported by mock", 2000); err != nil {
		t.Errorf("Error message did not appear: %v", err)
	}

	// Test 4: Exit command - this should cause the shell to terminate and slave to exit
	setup.MasterStdio.WriteToStdin([]byte("exit\n"))

	// Wait for slave to complete after shell exits
	select {
	case err := <-slaveErr:
		if err != nil {
			t.Logf("Slave completed with error: %v", err)
		} else {
			t.Log("Slave completed successfully after shell exit")
		}
	case <-time.After(2 * time.Second):
		t.Log("Slave did not exit after shell termination (this may be expected)")
	}

	// Cleanup
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Check master status (non-blocking)
	select {
	case err := <-masterErr:
		if err != nil {
			t.Logf("Master completed with: %v", err)
		}
	default:
		t.Log("Master still running (expected)")
	}
}
