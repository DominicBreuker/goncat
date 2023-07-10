package pipeio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"syscall"

	"github.com/muesli/cancelreader"
)

// Pipe ...
func Pipe(ctx context.Context, rwc1 io.ReadWriteCloser, rwc2 io.ReadWriteCloser, logfunc func(error)) {
	var wg sync.WaitGroup
	var o sync.Once

	stopCh := make(chan struct{})
	closed := false
	close := func() {
		closed = true

		rwc1.Close()
		rwc2.Close()

		close(stopCh)

		wg.Done()
	}
	wg.Add(1)

	go func() {
		select {
		case <-stopCh:
		case <-ctx.Done():
			o.Do(close)
		}
	}()

	go func() {
		var err error
		_, err = io.Copy(rwc1, rwc2)
		if err != nil {
			if !closed && !errors.Is(err, cancelreader.ErrCanceled) && !errors.Is(err, syscall.ECONNRESET) {
				logfunc(fmt.Errorf("io.Copy(rwc1, rwc2): %s", err))
			}
		}

		o.Do(close)
	}()

	go func() {
		var err error
		_, err = io.Copy(rwc2, rwc1)
		if err != nil {
			if !closed && !errors.Is(err, cancelreader.ErrCanceled) && !errors.Is(err, syscall.ECONNRESET) {
				logfunc(fmt.Errorf("io.Copy(rwc2, rwc1): %s", err))
			}
		}

		o.Do(close)
	}()

	wg.Wait()
}
