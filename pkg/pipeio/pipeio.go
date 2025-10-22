package pipeio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"

	"github.com/muesli/cancelreader"
)

func isBenignCopyErr(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	if errors.Is(err, cancelreader.ErrCanceled) {
		return true
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	// syscall-level “normal” disconnects
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	// timeouts are expected in some shutdown paths
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	// Go stdlib sometimes surfaces this string
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

func Pipe(ctx context.Context, rwc1 io.ReadWriteCloser, rwc2 io.ReadWriteCloser, logfunc func(error)) error {
	var once sync.Once
	var firstErr error
	var firstErrMu sync.Mutex

	shutdown := func() {
		_ = rwc1.Close()
		_ = rwc2.Close()
	}

	setErr := func(err error) {
		if err == nil || isBenignCopyErr(err) {
			return
		}
		firstErrMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		firstErrMu.Unlock()
	}

	copyOne := func(dst, src io.ReadWriteCloser) {
		// Optional: if you ever want to bound reads when ctx has a deadline:
		// if d, ok := ctx.Deadline(); ok { _ = setReadDeadlineIfPossible(src, d) }
		_, err := io.Copy(dst, src)
		if !isBenignCopyErr(err) {
			logfunc(fmt.Errorf("io.Copy: %w", err))
		}
		setErr(err)
		once.Do(shutdown)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); copyOne(rwc1, rwc2) }()
	go func() { defer wg.Done(); copyOne(rwc2, rwc1) }()

	// Cancel shuts both sides; Copy goroutines will exit.
	go func() {
		<-ctx.Done()
		once.Do(shutdown)
	}()

	wg.Wait()
	firstErrMu.Lock()
	defer firstErrMu.Unlock()
	return firstErr
}
