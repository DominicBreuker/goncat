package pipeio

import (
	"bytes"
	"dominicbreuker/goncat/mocks"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/muesli/cancelreader"
)

func TestNewStdio(t *testing.T) {
	t.Parallel()

	stdio := NewStdio(nil, nil)

	if stdio == nil {
		t.Fatal("NewStdio() returned nil")
	}
	if stdio.stdin == nil {
		t.Error("NewStdio() stdin is nil")
	}
	if stdio.stdout == nil {
		t.Error("NewStdio() stdout is nil")
	}
	// cancellableStdin may be nil on platforms that don't support it,
	// but that's acceptable
}

func TestStdio_Close(t *testing.T) {
	t.Parallel()

	stdio := NewStdio(nil, nil)

	if err := stdio.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestStdio_Read(t *testing.T) {
	t.Parallel()

	// Create simple test stdin using bytes.Buffer
	testData := []byte("test input")
	stdin := bytes.NewReader(testData)

	stdio := &Stdio{
		stdin:  stdin,
		stdout: io.Discard,
	}

	buf := make([]byte, 1024)
	n, err := stdio.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Read() read %d bytes, want %d", n, len(testData))
	}
	if !bytes.Equal(buf[:n], testData) {
		t.Errorf("Read() = %q, want %q", buf[:n], testData)
	}
}

func TestStdio_ReadWithCancellable(t *testing.T) {
	t.Parallel()

	// Create a Stdio with a cancellable stdin
	testData := []byte("test input")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Create cancelable reader
	cr, err := cancelreader.NewReader(r)
	if err != nil {
		t.Skipf("Cannot create cancelreader on this platform: %v", err)
	}

	stdio := &Stdio{
		stdin:            r,
		cancellableStdin: cr,
		stdout:           os.Stdout,
	}

	// Write test data
	go func() {
		w.Write(testData)
		w.Close()
	}()

	buf := make([]byte, 1024)
	n, err := stdio.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Read() read %d bytes, want %d", n, len(testData))
	}
	if !bytes.Equal(buf[:n], testData) {
		t.Errorf("Read() = %q, want %q", buf[:n], testData)
	}
}

func TestStdio_Write(t *testing.T) {
	t.Parallel()

	// Use mock stdio for testing - directly construct Stdio to avoid cancelreader issues
	mockStdio := mocks.NewMockStdio()
	defer mockStdio.Close()

	stdio := &Stdio{
		stdin:  mockStdio.GetStdin(),
		stdout: mockStdio.GetStdout(),
	}

	testData := []byte("test output")
	n, err := stdio.Write(testData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(testData))
	}

	// Wait for output to be captured
	if err := mockStdio.WaitForOutput(string(testData), 1000); err != nil {
		t.Fatalf("WaitForOutput() error = %v", err)
	}

	// Verify what was written to stdout
	output := mockStdio.ReadFromStdout()
	if !strings.Contains(output, string(testData)) {
		t.Errorf("Write() wrote %q, want to contain %q", output, string(testData))
	}
}

func TestStdio_CloseWithCancellable(t *testing.T) {
	t.Parallel()

	// Create a pipe for testing
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Create cancelable reader
	cr, err := cancelreader.NewReader(r)
	if err != nil {
		t.Skipf("Cannot create cancelreader on this platform: %v", err)
	}

	stdio := &Stdio{
		stdin:            r,
		cancellableStdin: cr,
		stdout:           os.Stdout,
	}

	// Close should cancel the reader
	if err := stdio.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// After cancellation, reads should return error
	buf := make([]byte, 10)
	_, err = stdio.Read(buf)
	if err == nil {
		t.Error("Expected error after Close(), got nil")
	}
}
