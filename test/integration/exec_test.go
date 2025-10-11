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

// TestExecCommandExecution simulates a complete master-slave connection
// with command execution, using mocked network, stdio, and exec.
// This test mimics the behavior of:
//   - "goncat master listen 'tcp://*:12345' --exec /bin/sh" (master listening with exec)
//   - "goncat slave connect tcp://127.0.0.1:12345" (slave connecting)
func TestExecCommandExecution(t *testing.T) {
	// Create mock network for TCP connections
	mockNet := mocks.NewMockTCPNetwork()

	// Create mock stdio for master and slave
	masterStdio := mocks.NewMockStdio()
	slaveStdio := mocks.NewMockStdio()
	defer masterStdio.Close()
	defer slaveStdio.Close()

	// Create mock exec for slave to simulate command execution
	mockExec := mocks.NewMockExec()

	// Setup master dependencies (network + stdio)
	masterDeps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
		Stdin:       func() io.Reader { return masterStdio.GetStdin() },
		Stdout:      func() io.Writer { return masterStdio.GetStdout() },
	}

	// Setup slave dependencies (network + stdio + exec)
	slaveDeps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
		Stdin:       func() io.Reader { return slaveStdio.GetStdin() },
		Stdout:      func() io.Writer { return slaveStdio.GetStdout() },
		ExecCommand: mockExec.Command,
	}

	// Master configuration - simulates "master listen 'tcp://*:12345' --exec /bin/sh"
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
		Exec:    "/bin/sh", // Execute /bin/sh on slave
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

	// Test exec functionality: master writes command input, should be echoed back
	// The mock exec echoes stdin to stdout, simulating a shell
	commandInput := "echo hello from shell\n"
	masterStdio.WriteToStdin([]byte(commandInput))

	// Wait for data to flow through the network to slave, get executed, and return
	time.Sleep(500 * time.Millisecond)

	// Verify the command output came back to master's stdout
	masterOutput := masterStdio.ReadFromStdout()
	if !strings.Contains(masterOutput, "echo hello from shell") {
		t.Errorf("Expected master stdout to contain command echo, got: %q", masterOutput)
	}

	// Test bidirectional communication with the executed command
	commandInput2 := "second command\n"
	masterStdio.WriteToStdin([]byte(commandInput2))
	time.Sleep(300 * time.Millisecond)

	masterOutput2 := masterStdio.ReadFromStdout()
	if !strings.Contains(masterOutput2, "second command") {
		t.Errorf("Expected master to receive second command echo, got: %q", masterOutput2)
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
