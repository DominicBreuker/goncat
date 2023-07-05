package pipeio

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/muesli/cancelreader"
)

// Pipe ...
func Pipe(rwc1 io.ReadWriteCloser, rwc2 io.ReadWriteCloser, logfunc func(error)) {
	var wg sync.WaitGroup
	var o sync.Once

	closed := false
	close := func() {
		closed = true

		rwc1.Close()
		rwc2.Close()

		wg.Done()
	}
	wg.Add(1)

	go func() {
		var err error
		_, err = io.Copy(rwc1, rwc2)
		if err != nil {
			if !closed && !errors.Is(err, cancelreader.ErrCanceled) {
				logfunc(fmt.Errorf("io.Copy(rwc1, rwc2): %s", err))
			}
		}

		o.Do(close)
	}()

	go func() {
		var err error
		_, err = io.Copy(rwc2, rwc1)
		if err != nil {
			if !closed && !errors.Is(err, cancelreader.ErrCanceled) {
				logfunc(fmt.Errorf("io.Copy(rwc2, rwc1): %s", err))
			}
		}

		o.Do(close)
	}()

	wg.Wait()
}
