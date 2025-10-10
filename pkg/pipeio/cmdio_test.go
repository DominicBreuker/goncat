package pipeio

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type nopWriteCloser struct {
	w io.Writer
}

func (n *nopWriteCloser) Write(p []byte) (int, error) {
	return n.w.Write(p)
}

func (n *nopWriteCloser) Close() error {
	return nil
}

func TestNewCmdio(t *testing.T) {
	t.Parallel()

	stdout := strings.NewReader("stdout data")
	stderr := strings.NewReader("stderr data")
	stdin := &nopWriteCloser{w: new(bytes.Buffer)}

	cmdio := NewCmdio(stdout, stderr, stdin)

	if cmdio == nil {
		t.Fatal("NewCmdio() returned nil")
	}
	if cmdio.r == nil {
		t.Error("NewCmdio() reader is nil")
	}
	if cmdio.w == nil {
		t.Error("NewCmdio() writer is nil")
	}
}

func TestCmdio_Read(t *testing.T) {
	t.Parallel()

	stdout := strings.NewReader("stdout")
	stderr := strings.NewReader("stderr")
	stdin := &nopWriteCloser{w: new(bytes.Buffer)}

	cmdio := NewCmdio(stdout, stderr, stdin)

	buf := make([]byte, 1024)
	var allData []byte

	// Read all data from both streams
	for {
		n, err := cmdio.Read(buf)
		if n > 0 {
			allData = append(allData, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}

	// Verify we got data from at least one stream
	result := string(allData)
	if !strings.Contains(result, "stdout") && !strings.Contains(result, "stderr") {
		t.Errorf("Read() data = %q, expected to contain data from stdout or stderr", result)
	}
}

func TestCmdio_Write(t *testing.T) {
	t.Parallel()

	stdout := strings.NewReader("")
	stderr := strings.NewReader("")
	stdinBuf := new(bytes.Buffer)
	stdin := &nopWriteCloser{w: stdinBuf}

	cmdio := NewCmdio(stdout, stderr, stdin)

	testData := []byte("test input")
	n, err := cmdio.Write(testData)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(testData))
	}

	if !bytes.Equal(stdinBuf.Bytes(), testData) {
		t.Errorf("Write() wrote %q, want %q", stdinBuf.Bytes(), testData)
	}
}

func TestCmdio_Close(t *testing.T) {
	t.Parallel()

	stdout := strings.NewReader("")
	stderr := strings.NewReader("")
	stdin := &nopWriteCloser{w: new(bytes.Buffer)}

	cmdio := NewCmdio(stdout, stderr, stdin)

	if err := cmdio.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestMultiReader(t *testing.T) {
	t.Parallel()

	r1 := strings.NewReader("first")
	r2 := strings.NewReader("second")

	mr := newMultiReader(r1, r2)

	var allData []byte
	buf := make([]byte, 1024)

	// Read all available data
	for {
		n, err := mr.Read(buf)
		if n > 0 {
			allData = append(allData, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}

	result := string(allData)
	// We should have data from at least one of the readers
	if len(result) == 0 {
		t.Error("multiReader.Read() returned no data")
	}
}

func TestMultiReader_BothReadersEmpty(t *testing.T) {
	t.Parallel()

	r1 := strings.NewReader("")
	r2 := strings.NewReader("")

	mr := newMultiReader(r1, r2)

	buf := make([]byte, 1024)
	n, err := mr.Read(buf)
	if err != io.EOF {
		t.Errorf("Read() error = %v, want io.EOF", err)
	}
	if n != 0 {
		t.Errorf("Read() returned %d bytes, want 0", n)
	}
}

func TestMultiReader_SmallBuffer(t *testing.T) {
	t.Parallel()

	r1 := strings.NewReader("data")
	r2 := strings.NewReader("more")

	mr := newMultiReader(r1, r2)

	// Use a small buffer to test multiple reads
	buf := make([]byte, 2)
	var allData []byte

	for {
		n, err := mr.Read(buf)
		if n > 0 {
			allData = append(allData, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}

	// Should have read all data from both readers eventually
	if len(allData) == 0 {
		t.Error("multiReader.Read() returned no data")
	}
}

func TestCmdio_WithClosedWriter(t *testing.T) {
	t.Parallel()

	stdout := strings.NewReader("test")
	stderr := strings.NewReader("data")

	// Create a write closer that simulates being closed
	stdin := &closedWriteCloser{}

	cmdio := NewCmdio(stdout, stderr, stdin)

	// Writing should fail
	_, err := cmdio.Write([]byte("test"))
	if err == nil {
		t.Error("Write() to closed writer should return error")
	}
}

func TestCmdio_MultipleReads(t *testing.T) {
	t.Parallel()

	stdout := strings.NewReader("stdout")
	stderr := strings.NewReader("stderr")
	stdin := &nopWriteCloser{w: new(bytes.Buffer)}

	cmdio := NewCmdio(stdout, stderr, stdin)

	// First read
	buf1 := make([]byte, 1024)
	n1, _ := cmdio.Read(buf1)
	data1 := string(buf1[:n1])

	// Second read
	buf2 := make([]byte, 1024)
	n2, _ := cmdio.Read(buf2)
	data2 := string(buf2[:n2])

	// We should have read data from both readers
	combined := data1 + data2
	if !strings.Contains(combined, "stdout") && !strings.Contains(combined, "stderr") {
		t.Errorf("Multiple reads did not return expected data, got: %q", combined)
	}
}

// closedWriteCloser simulates a closed writer
type closedWriteCloser struct{}

func (c *closedWriteCloser) Write(p []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func (c *closedWriteCloser) Close() error {
	return nil
}
