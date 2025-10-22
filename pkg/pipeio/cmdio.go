package pipeio

import (
	"io"
	"sync"
)

type Cmdio struct {
	r io.Reader
	w io.WriteCloser
}

func NewCmdio(stdout, stderr io.Reader, stdin io.WriteCloser) *Cmdio {
	return &Cmdio{
		r: newMultiReader(stdout, stderr),
		w: stdin,
	}
}

func (s *Cmdio) Read(p []byte) (int, error)  { return s.r.Read(p) }
func (s *Cmdio) Write(p []byte) (int, error) { return s.w.Write(p) }
func (s *Cmdio) Close() error                { return s.w.Close() }

type multiReader struct {
	dataCh chan []byte
	closed chan struct{}
	wg     sync.WaitGroup
}

func newMultiReader(r1, r2 io.Reader) *multiReader {
	mr := &multiReader{
		dataCh: make(chan []byte, 8),
		closed: make(chan struct{}),
	}
	mr.wg.Add(2)
	go mr.readFrom(r1)
	go mr.readFrom(r2)
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
			// copy because buf is reused
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case mr.dataCh <- chunk:
			case <-mr.closed:
				return
			}
		}
		if err != nil {
			return // EOF or other error; we just stop this branch
		}
	}
}

func (mr *multiReader) Read(p []byte) (int, error) {
	data, ok := <-mr.dataCh
	if !ok {
		return 0, io.EOF
	}
	return copy(p, data), nil
}
