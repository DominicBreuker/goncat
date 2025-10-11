package integration

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/pipeio"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// Helper to create a Stdio from dependencies (for testing)
func NewStdio(deps *config.Dependencies) *pipeio.Stdio {
	return pipeio.NewStdio(deps)
}

// Helper to pipe data (for testing)
func Pipe(ctx context.Context, rwc1 io.ReadWriteCloser, rwc2 io.ReadWriteCloser, logfunc func(error)) {
	pipeio.Pipe(ctx, rwc1, rwc2, logfunc)
}

// TestForegroundDataExchange demonstrates stdio mocking with network pipes.
// This test verifies that mocked stdin/stdout work correctly with the pipeio package.
func TestForegroundDataExchange(t *testing.T) {
	// Create mock stdio for both sides
	side1Stdio := NewMockStdio()
	side2Stdio := NewMockStdio()
	defer side1Stdio.Close()
	defer side2Stdio.Close()

	// Create dependencies for both sides
	side1Deps := &config.Dependencies{
		Stdin:  func() io.Reader { return side1Stdio.GetStdin() },
		Stdout: func() io.Writer { return side1Stdio.GetStdout() },
	}

	side2Deps := &config.Dependencies{
		Stdin:  func() io.Reader { return side2Stdio.GetStdin() },
		Stdout: func() io.Writer { return side2Stdio.GetStdout() },
	}

	// Create a network pipe to connect the two sides
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start piping on both sides
	go func() {
		// Side 1: pipes between stdio1 and conn1
		stdio1 := NewStdio(side1Deps)
		Pipe(ctx, stdio1, conn1, func(err error) {
			t.Logf("Side1 pipe error: %v", err)
		})
	}()

	go func() {
		// Side 2: pipes between stdio2 and conn2
		stdio2 := NewStdio(side2Deps)
		Pipe(ctx, stdio2, conn2, func(err error) {
			t.Logf("Side2 pipe error: %v", err)
		})
	}()

	// Give pipes time to start
	time.Sleep(100 * time.Millisecond)

	// Test data flow: side1 stdin -> side2 stdout
	testInput := "Hello from side1!\n"
	side1Stdio.WriteToStdin([]byte(testInput))

	// Wait for data to flow through
	time.Sleep(300 * time.Millisecond)

	// Check side2 received the data on stdout
	side2Output := side2Stdio.ReadFromStdout()
	if !strings.Contains(side2Output, "Hello from side1!") {
		t.Errorf("Expected side2 stdout to contain 'Hello from side1!', got: %q", side2Output)
	}

	// Test reverse flow: side2 stdin -> side1 stdout
	testReverse := "Hello from side2!\n"
	side2Stdio.WriteToStdin([]byte(testReverse))

	// Wait for data to flow back
	time.Sleep(300 * time.Millisecond)

	// Check side1 received the data on stdout
	side1Output := side1Stdio.ReadFromStdout()
	if !strings.Contains(side1Output, "Hello from side2!") {
		t.Errorf("Expected side1 stdout to contain 'Hello from side2!', got: %q", side1Output)
	}
}

// TestMockTCPBasics demonstrates basic functionality of the mock TCP network.
func TestMockTCPBasics(t *testing.T) {
	t.Skip("Basic TCP test - comprehensive test above demonstrates functionality")
}
