package exec

import (
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/pipeio"
	"fmt"
	"io"
	"net"
	"os/exec"
	"sync"
)

// Run ...
func Run(conn net.Conn, program string) error {
	cmd := exec.Command(program)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdoutPipe(): %s", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdinPipe(): %s", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cmd.StderrPipe(): %s", err)
	}

	rwc := NewCmdIO(NewMultiReader(stdout, stderr), stdin)

	//cmd.Stdout = conn
	//cmd.Stdin = conn
	//cmd.Stderr = conn

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Run(): %s", err)
	}

	done := make(chan struct{})

	go func() {
		cmd.Wait()
		done <- struct{}{}
	}()

	go func() {
		pipeio.Pipe(rwc, conn, func(err error) {
			log.ErrorMsg("Run Pipe(pty, conn): %s\n", err)
		})
		cmd.Process.Kill()
		done <- struct{}{}
	}()
	<-done

	//cmd.Wait() // TODO: find out how to wait just for exit, not for IO to finish. this hangs if you exit a non-PTY shell until you try to send data

	return nil
}

type CmdIO struct {
	stdin  io.Reader
	stdout io.WriteCloser
}

// NewStdio sets up a new Stdio with cancellable reader in stdin if supported
//func NewStdio() *Stdio {
func NewCmdIO(r io.Reader, w io.WriteCloser) *CmdIO {
	out := CmdIO{
		stdin:  r,
		stdout: w,
	}
	return &out
}

// Read reads from stdin
func (s *CmdIO) Read(p []byte) (n int, err error) {
	return s.stdin.Read(p)
}

// Write writes to stdout
func (s *CmdIO) Write(p []byte) (n int, err error) {
	return s.stdout.Write(p)
}

// Close cancels reads from stdin if possible
func (s *CmdIO) Close() error {
	return s.stdout.Close()
}

// #########

type MultiReader struct {
	reader1 io.Reader
	reader2 io.Reader
	dataCh  chan []byte
	errCh   chan error
	once    sync.Once
}

func NewMultiReader(reader1, reader2 io.Reader) *MultiReader {
	mr := &MultiReader{
		reader1: reader1,
		reader2: reader2,
		dataCh:  make(chan []byte),
		errCh:   make(chan error, 2), // buffer of size 2 for errors from both readers
	}

	go mr.readFromReader(reader1)
	go mr.readFromReader(reader2)

	return mr
}

func (mr *MultiReader) readFromReader(reader io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
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

func (mr *MultiReader) Read(p []byte) (n int, err error) {
	mr.once.Do(func() {
		go func() {
			var errCount int
			for err := range mr.errCh {
				fmt.Printf("DEBUG: err= %s\n", err)
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
