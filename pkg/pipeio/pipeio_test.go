package pipeio

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/muesli/cancelreader"
)

// fakeRWC is a fake ReadWriteCloser for testing.
type fakeRWC struct {
	reader io.Reader
	writer io.Writer
	closed bool
	mu     sync.Mutex
}

func newFakeRWC(reader io.Reader, writer io.Writer) *fakeRWC {
	return &fakeRWC{
		reader: reader,
		writer: writer,
	}
}

func (f *fakeRWC) Read(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, io.EOF
	}
	return f.reader.Read(p)
}

func (f *fakeRWC) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, io.ErrClosedPipe
	}
	return f.writer.Write(p)
}

func (f *fakeRWC) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func TestPipe_BasicBidirectionalCopy(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create two pipe pairs for bidirectional communication
	client1, server1 := net.Pipe()
	defer client1.Close()
	defer server1.Close()

	// Track errors
	var loggedErrors []error
	var mu sync.Mutex
	logFunc := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		loggedErrors = append(loggedErrors, err)
	}

	// Start Pipe in a goroutine
	done := make(chan struct{})
	go func() {
		Pipe(ctx, client1, server1, logFunc)
		close(done)
	}()

	// Write from client1 and read from server1
	testData := []byte("hello from client")
	go func() {
		client1.Write(testData)
	}()

	buf := make([]byte, 1024)
	n, err := server1.Read(buf)
	if err != nil {
		t.Fatalf("server1.Read() error = %v", err)
	}
	if string(buf[:n]) != string(testData) {
		t.Errorf("server1.Read() = %q, want %q", string(buf[:n]), string(testData))
	}

	// Write from server1 and read from client1
	responseData := []byte("hello from server")
	go func() {
		server1.Write(responseData)
	}()

	n, err = client1.Read(buf)
	if err != nil {
		t.Fatalf("client1.Read() error = %v", err)
	}
	if string(buf[:n]) != string(responseData) {
		t.Errorf("client1.Read() = %q, want %q", string(buf[:n]), string(responseData))
	}

	// Cancel context to stop Pipe
	cancel()

	select {
	case <-done:
		// Success - Pipe returned
	case <-time.After(1 * time.Second):
		t.Error("Pipe() did not return after context cancellation")
	}
}

func TestPipe_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	// Use net.Pipe for realistic connection behavior
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var loggedErrors []error
	var mu sync.Mutex
	logFunc := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		loggedErrors = append(loggedErrors, err)
	}

	// Start Pipe in a goroutine
	done := make(chan struct{})
	go func() {
		Pipe(ctx, client, server, logFunc)
		close(done)
	}()

	// Cancel context immediately
	cancel()

	// Wait for Pipe to return
	select {
	case <-done:
		// Success - Pipe returned
	case <-time.After(2 * time.Second):
		t.Error("Pipe() did not return after context cancellation")
	}
}

func TestPipe_EOF(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Use strings.Reader which will return EOF immediately
	reader1 := strings.NewReader("")
	reader2 := strings.NewReader("")

	rwc1 := newFakeRWC(reader1, io.Discard)
	rwc2 := newFakeRWC(reader2, io.Discard)

	var loggedErrors []error
	var mu sync.Mutex
	logFunc := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		loggedErrors = append(loggedErrors, err)
	}

	// Pipe should return when EOF is encountered
	done := make(chan struct{})
	go func() {
		Pipe(ctx, rwc1, rwc2, logFunc)
		close(done)
	}()

	select {
	case <-done:
		// Success - Pipe returned on EOF
	case <-time.After(2 * time.Second):
		t.Error("Pipe() did not return on EOF")
	}

	// Verify both were closed
	if !rwc1.closed || !rwc2.closed {
		t.Error("Pipe() did not close both connections")
	}
}

func TestPipe_IgnoresCancelReaderError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create a reader that returns cancelreader.ErrCanceled
	errorReader := &errorReader{err: cancelreader.ErrCanceled}

	rwc1 := newFakeRWC(errorReader, io.Discard)
	rwc2 := newFakeRWC(strings.NewReader(""), io.Discard)

	var loggedErrors []error
	var mu sync.Mutex
	logFunc := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		loggedErrors = append(loggedErrors, err)
	}

	done := make(chan struct{})
	go func() {
		Pipe(ctx, rwc1, rwc2, logFunc)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Pipe() did not return after cancelreader error")
	}

	// Verify cancelreader error was NOT logged
	mu.Lock()
	defer mu.Unlock()
	for _, err := range loggedErrors {
		if errors.Is(err, cancelreader.ErrCanceled) {
			t.Error("cancelreader.ErrCanceled should not be logged")
		}
	}
}

func TestPipe_IgnoresConnectionResetError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create a reader that returns syscall.ECONNRESET
	errorReader := &errorReader{err: syscall.ECONNRESET}

	rwc1 := newFakeRWC(errorReader, io.Discard)
	rwc2 := newFakeRWC(strings.NewReader(""), io.Discard)

	var loggedErrors []error
	var mu sync.Mutex
	logFunc := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		loggedErrors = append(loggedErrors, err)
	}

	done := make(chan struct{})
	go func() {
		Pipe(ctx, rwc1, rwc2, logFunc)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Pipe() did not return after connection reset")
	}

	// Verify ECONNRESET error was NOT logged
	mu.Lock()
	defer mu.Unlock()
	for _, err := range loggedErrors {
		if errors.Is(err, syscall.ECONNRESET) {
			t.Error("syscall.ECONNRESET should not be logged")
		}
	}
}

func TestPipe_ClosesBothConnections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	rwc1 := newFakeRWC(strings.NewReader(""), io.Discard)
	rwc2 := newFakeRWC(strings.NewReader(""), io.Discard)

	logFunc := func(err error) {}

	done := make(chan struct{})
	go func() {
		Pipe(ctx, rwc1, rwc2, logFunc)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Pipe() did not return")
	}

	// Verify both connections were closed
	if !rwc1.closed {
		t.Error("rwc1 was not closed")
	}
	if !rwc2.closed {
		t.Error("rwc2 was not closed")
	}
}

// errorReader is a reader that returns a specific error immediately.
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
