// Package helpers provides common utilities for integration and end-to-end tests.
package helpers

import (
	"context"
	"dominicbreuker/goncat/mocks"
	mocks_tcp "dominicbreuker/goncat/mocks/tcp"
	"dominicbreuker/goncat/pkg/config"
	"io"
	"net"
	"time"
)

// mockUDPDialer creates a mock UDP dialer that uses the mock UDP network.
// It returns a function that creates a connected UDP socket using the mock network.
func mockUDPDialer(mockNet *mocks.MockUDPNetwork) config.UDPDialerFunc {
	return func(ctx context.Context, network string, laddr, raddr *net.UDPAddr) (net.PacketConn, error) {
		// Create a local listener with an ephemeral port to act as the "dialed" connection
		conn, err := mockNet.ListenUDP(network, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		if err != nil {
			return nil, err
		}
		// Note: This creates an "unconnected" UDP socket, but the real implementation
		// uses net.Dial which creates a "connected" one. For our mock purposes,
		// the difference doesn't matter as much since we control both ends.
		return conn, nil
	}
}

// MockDependenciesAndConfigs contains all mock dependencies and default configurations
// for integration tests. It provides a unified setup that can be easily customized
// for specific test scenarios.
type MockDependenciesAndConfigs struct {
	// Mock networks
	TCPNetwork *mocks_tcp.MockTCPNetwork
	UDPNetwork *mocks.MockUDPNetwork

	// Mock stdio for master and slave
	MasterStdio *mocks.MockStdio
	SlaveStdio  *mocks.MockStdio

	// Mock exec for command execution tests
	MockExec *mocks.MockExec

	// Dependencies for master and slave
	MasterDeps *config.Dependencies
	SlaveDeps  *config.Dependencies

	// Default configurations (can be modified by tests)
	MasterSharedCfg *config.Shared
	SlaveSharedCfg  *config.Shared
	MasterCfg       *config.Master
}

// SetupMockDependenciesAndConfigs creates a complete set of mock dependencies
// and default configurations for integration tests. This provides a unified
// setup that reduces boilerplate in tests.
//
// Usage:
//
//	setup := helpers.SetupMockDependenciesAndConfigs()
//	defer setup.Close()
//	// Modify configs as needed for the specific test
//	setup.MasterCfg.Exec = "/bin/sh"
func SetupMockDependenciesAndConfigs() *MockDependenciesAndConfigs {
	// Create mock networks
	mockTCPNet := mocks_tcp.NewMockTCPNetwork()
	mockUDPNet := mocks.NewMockUDPNetwork()

	// Create mock stdio for master and slave
	masterStdio := mocks.NewMockStdio()
	slaveStdio := mocks.NewMockStdio()

	// Create mock exec
	mockExec := mocks.NewMockExec()

	// Setup master dependencies
	masterDeps := &config.Dependencies{
		TCPDialer:      mockTCPNet.DialTCPContext,
		TCPListener:    mockTCPNet.ListenTCP,
		UDPDialer:      mockUDPDialer(mockUDPNet),
		UDPListener:    mockUDPNet.ListenUDP,
		PacketListener: mockUDPNet.ListenPacket,
		Stdin:          func() io.Reader { return masterStdio.GetStdin() },
		Stdout:         func() io.Writer { return masterStdio.GetStdout() },
		ExecCommand:    mockExec.Command,
	}

	// Setup slave dependencies
	slaveDeps := &config.Dependencies{
		TCPDialer:      mockTCPNet.DialTCPContext,
		TCPListener:    mockTCPNet.ListenTCP,
		UDPDialer:      mockUDPDialer(mockUDPNet),
		UDPListener:    mockUDPNet.ListenUDP,
		PacketListener: mockUDPNet.ListenPacket,
		Stdin:          func() io.Reader { return slaveStdio.GetStdin() },
		Stdout:         func() io.Writer { return slaveStdio.GetStdout() },
		ExecCommand:    mockExec.Command,
	}

	// Create default shared config for master
	masterSharedCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Timeout:  10 * time.Second,
		Deps:     masterDeps,
	}

	// Create default shared config for slave
	slaveSharedCfg := &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Timeout:  10 * time.Second,
		Deps:     slaveDeps,
	}

	// Create default master config
	masterCfg := &config.Master{
		Exec:    "",
		Pty:     false,
		LogFile: "",
	}

	return &MockDependenciesAndConfigs{
		TCPNetwork:      mockTCPNet,
		UDPNetwork:      mockUDPNet,
		MasterStdio:     masterStdio,
		SlaveStdio:      slaveStdio,
		MockExec:        mockExec,
		MasterDeps:      masterDeps,
		SlaveDeps:       slaveDeps,
		MasterSharedCfg: masterSharedCfg,
		SlaveSharedCfg:  slaveSharedCfg,
		MasterCfg:       masterCfg,
	}
}

// Close cleans up all mock resources. Should be called with defer.
func (m *MockDependenciesAndConfigs) Close() {
	if m.MasterStdio != nil {
		m.MasterStdio.Close()
	}
	if m.SlaveStdio != nil {
		m.SlaveStdio.Close()
	}
}
