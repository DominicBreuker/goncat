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
type multiReader struct {
	r1 io.Reader
	r2 io.Reader

	dataCh chan []byte
	errCh  chan error
	once   sync.Once
}

// newMultiReader creates a new multiReader that reads from both r1 and r2 concurrently.
func newMultiReader(r1, r2 io.Reader) *multiReader {
	mr := &multiReader{
		r1: r1,
		r2: r2,

		dataCh: make(chan []byte),
		errCh:  make(chan error, 2), // buffer for errors from both readers
	}

	go mr.readFrom(r1)
	go mr.readFrom(r2)

	return mr
}

func (mr *multiReader) readFrom(r io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := r.Read(buffer)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			mr.dataCh <- data
		}
		if err != nil {
			mr.errCh <- err
			return
		}
	}
}

func (mr *multiReader) Read(p []byte) (n int, err error) {
	mr.once.Do(func() {
		go func() {
			var errCount int
			for range mr.errCh {
				errCount++
				if errCount == 2 {
					close(mr.dataCh)
				}
			}
		}()
	})

	data, ok := <-mr.dataCh
	if !ok {
		return 0, io.EOF
	}

	n = copy(p, data)
	return n, nil
}
