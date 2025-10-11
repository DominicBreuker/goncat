// Package helpers provides common utilities for integration and end-to-end tests.
package helpers

import (
	"dominicbreuker/goncat/mocks"
	"dominicbreuker/goncat/pkg/config"
	"io"
)

// SetupMockDependencies creates a complete set of mock dependencies
// for testing with both mocked network and stdio.
func SetupMockDependencies() (*mocks.MockTCPNetwork, *mocks.MockStdio, *config.Dependencies) {
	mockNet := mocks.NewMockTCPNetwork()
	mockStdio := mocks.NewMockStdio()

	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
		Stdin:       func() io.Reader { return mockStdio.GetStdin() },
		Stdout:      func() io.Writer { return mockStdio.GetStdout() },
	}

	return mockNet, mockStdio, deps
}

// SetupMockDependenciesWithExec creates mock dependencies including exec command mocking.
// This is useful for testing command execution scenarios.
func SetupMockDependenciesWithExec() (*mocks.MockTCPNetwork, *mocks.MockStdio, *mocks.MockExec, *config.Dependencies) {
	mockNet := mocks.NewMockTCPNetwork()
	mockStdio := mocks.NewMockStdio()
	mockExec := mocks.NewMockExec()

	deps := &config.Dependencies{
		TCPDialer:   mockNet.DialTCP,
		TCPListener: mockNet.ListenTCP,
		Stdin:       func() io.Reader { return mockStdio.GetStdin() },
		Stdout:      func() io.Writer { return mockStdio.GetStdout() },
		ExecCommand: mockExec.Command,
	}

	return mockNet, mockStdio, mockExec, deps
}

// DefaultSharedConfig creates a default Shared configuration for testing.
func DefaultSharedConfig(deps *config.Dependencies) *config.Shared {
	return &config.Shared{
		Protocol: config.ProtoTCP,
		Host:     "127.0.0.1",
		Port:     12345,
		SSL:      false,
		Key:      "",
		Verbose:  false,
		Deps:     deps,
	}
}

// DefaultMasterConfig creates a default Master configuration for testing.
func DefaultMasterConfig() *config.Master {
	return &config.Master{
		Exec:    "",
		Pty:     false,
		LogFile: "",
	}
}
