// Package semaphore provides a timeout-aware semaphore implementation
// for controlling concurrent connections with proper timeout handling.
package semaphore

import (
	"context"
	"fmt"
	"time"
)

// ConnSemaphore controls concurrent access with timeout support.
// It uses a buffered channel to limit the number of concurrent operations.
type ConnSemaphore struct {
	sem     chan struct{}
	timeout time.Duration
}

// New creates a semaphore with capacity n and default timeout.
// The semaphore starts with all n slots available.
func New(n int, timeout time.Duration) *ConnSemaphore {
	sem := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		sem <- struct{}{}
	}
	return &ConnSemaphore{sem: sem, timeout: timeout}
}

// Acquire attempts to acquire the semaphore within the timeout period.
// Returns error if timeout expires or context is cancelled.
// If the semaphore is nil, this is a no-op and returns nil.
func (s *ConnSemaphore) Acquire(ctx context.Context) error {
	if s == nil {
		return nil // no-op if semaphore not provided
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	select {
	case <-s.sem:
		return nil
	case <-timeoutCtx.Done():
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("timeout acquiring connection slot after %v", s.timeout)
	}
}

// Release releases the semaphore slot.
// If the semaphore is nil, this is a no-op.
func (s *ConnSemaphore) Release() {
	if s == nil {
		return // no-op if semaphore not provided
	}
	s.sem <- struct{}{}
}
