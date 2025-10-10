// Package pipeio provides utilities for bidirectional I/O piping between
// ReadWriteClosers, with support for context cancellation and error handling.
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

// Pipe establishes bidirectional I/O between two ReadWriteClosers.
// It copies data in both directions concurrently until one side closes,
// an error occurs, or the context is cancelled. The logfunc is called
// for non-fatal errors during copying.
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
