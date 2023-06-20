package pipeio

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Pipe ...
func Pipe(rwc1 io.ReadWriteCloser, rwc2 io.ReadWriteCloser, logfunc func(error)) {
	var wg sync.WaitGroup
	var o sync.Once

	close := func() {
		rwc1.Close()
		rwc2.Close()

		wg.Done()
	}
	wg.Add(1)

	go func() {
		var err error
		_, err = io.Copy(rwc1, rwc2)
		if err != nil {
			logfunc(fmt.Errorf("io.Copy(rwc1, rwc2): %s", err))
		}

		o.Do(close)
	}()

	go func() {
		var err error
		_, err = io.Copy(rwc2, rwc1)
		if err != nil {
			logfunc(fmt.Errorf("io.Copy(rwc2, rwc1): %s", err))
		}

		o.Do(close)
	}()

	wg.Wait()
}

// Stdio as a ReadWriteCloser
var Stdio = &struct {
	io.ReadCloser
	io.Writer
}{
	io.NopCloser(os.Stdin),
	os.Stdout,
}
