package pipeio

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/muesli/cancelreader"
)

func TestNewStdio(t *testing.T) {
	t.Parallel()

	stdio := NewStdio()

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

	stdio := NewStdio()

	if err := stdio.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestStdio_Read(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that interacts with stdin in short mode")
	}
	t.Parallel()

	// Create a Stdio with a fake stdin for testing
	testData := []byte("test input")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Write test data
	go func() {
		w.Write(testData)
		w.Close()
	}()

	stdio := &Stdio{
		stdin:  r,
		stdout: os.Stdout,
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
	if testing.Short() {
		t.Skip("skipping test that interacts with stdin in short mode")
	}
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
	if testing.Short() {
		t.Skip("skipping test that interacts with stdout in short mode")
	}
	t.Parallel()

	// Create a Stdio with a fake stdout for testing
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer r.Close()
	defer w.Close()

	stdio := &Stdio{
		stdin:  os.Stdin,
		stdout: w,
	}

	testData := []byte("test output")
	n, err := stdio.Write(testData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(testData))
	}

	// Close writer to signal EOF to reader
	w.Close()

	// Read back what was written
	buf := make([]byte, 1024)
	readN, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("r.Read() error = %v", err)
	}
	if readN != len(testData) {
		t.Errorf("r.Read() read %d bytes, want %d", readN, len(testData))
	}
	if !bytes.Equal(buf[:readN], testData) {
		t.Errorf("Write() wrote %q, want %q", buf[:readN], testData)
	}
}

func TestStdio_CloseWithCancellable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses cancelreader in short mode")
	}
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
