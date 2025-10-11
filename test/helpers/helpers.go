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
