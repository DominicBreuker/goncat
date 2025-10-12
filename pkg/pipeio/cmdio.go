package pipeio

import (
	"io"
	"sync"
)

// Cmdio wraps stdout/stderr readers and stdin writer for a command,
// providing a unified ReadWriteCloser interface.
type Cmdio struct {
	r io.Reader
	w io.WriteCloser
}

// NewCmdio creates a new Cmdio that combines stdout and stderr into a single reader
// and provides access to stdin as a writer. The stdout and stderr streams are
// multiplexed together for reading.
func NewCmdio(stdout, stderr io.Reader, stdin io.WriteCloser) *Cmdio {
	return &Cmdio{
		r: newMultiReader(stdout, stderr),
		w: stdin,
	}
}

// Read reads from the combined stdout/stderr stream.
func (s *Cmdio) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}

// Write writes to the command's stdin.
func (s *Cmdio) Write(p []byte) (n int, err error) {
	return s.w.Write(p)
}

// Close closes the stdin writer.
func (s *Cmdio) Close() error {
	return s.w.Close()
}

// multiReader multiplexes data from two readers into a single reader.
// It concurrently reads from both readers and provides data through Read calls.
// Reading continues until both readers return an error (typically io.EOF).
// Data from both readers is interleaved in the order it becomes available.
type multiReader struct {
	dataCh chan []byte
	errCh  chan error
	closed chan struct{}
	wg     sync.WaitGroup
}

func newMultiReader(r1, r2 io.Reader) *multiReader {
	mr := &multiReader{
		dataCh: make(chan []byte, 8), // buffered to avoid blocking
		errCh:  make(chan error, 2),  // buffered for both EOFs
		closed: make(chan struct{}),
	}

	// Start readers
	mr.wg.Add(2)
	go mr.readFrom(r1)
	go mr.readFrom(r2)

	// Close dataCh when both readers finish
	go func() {
		mr.wg.Wait()
		close(mr.dataCh)
		close(mr.closed)
	}()

	return mr
}

func (mr *multiReader) readFrom(r io.Reader) {
	defer mr.wg.Done()
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			// It's important to copy the buffer, since buf is reused
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case mr.dataCh <- data:
			case <-mr.closed:
				return
			}
		}
		if err != nil {
			// Signal error and stop reading
			return
		}
	}
}

// Read pulls data from either reader, interleaved in the order it arrives.
func (mr *multiReader) Read(p []byte) (n int, err error) {
	data, ok := <-mr.dataCh
	if !ok {
		return 0, io.EOF
	}
	n = copy(p, data)
	return n, nil
}
