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
func Pipe(ctx context.Context, rwc1 io.ReadWriteCloser, rwc2 io.ReadWriteCloser, logfunc func(error)) error {
	var once sync.Once
	var firstErr error
	var firstErrMu sync.Mutex

	shutdown := func() {
		_ = rwc1.Close()
		_ = rwc2.Close()
	}

	setErr := func(err error) {
		firstErrMu.Lock()
		defer firstErrMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	copyFunc := func(dst, src io.ReadWriteCloser) {
		_, err := io.Copy(dst, src)
		if err != nil && !errors.Is(err, cancelreader.ErrCanceled) && !errors.Is(err, syscall.ECONNRESET) {
			logfunc(fmt.Errorf("io.Copy: %w", err))
			setErr(err)
		}
		once.Do(shutdown)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		copyFunc(rwc1, rwc2)
	}()
	go func() {
		defer wg.Done()
		copyFunc(rwc2, rwc1)
	}()

	go func() {
		select {
		case <-ctx.Done():
			once.Do(shutdown)
		}
	}()

	wg.Wait()
	firstErrMu.Lock()
	defer firstErrMu.Unlock()
	return firstErr
}
