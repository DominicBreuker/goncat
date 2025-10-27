package semaphore

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	sem := New(5, 10*time.Second)
	if sem == nil {
		t.Fatal("New() returned nil")
	}
	if sem.timeout != 10*time.Second {
		t.Errorf("timeout = %v; want 10s", sem.timeout)
	}
	if cap(sem.sem) != 5 {
		t.Errorf("capacity = %d; want 5", cap(sem.sem))
	}
	if len(sem.sem) != 5 {
		t.Errorf("initial length = %d; want 5", len(sem.sem))
	}
}

func TestAcquireRelease(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		capacity int
		timeout  time.Duration
	}{
		{"capacity-1", 1, 1 * time.Second},
		{"capacity-5", 5, 1 * time.Second},
		{"capacity-100", 100, 1 * time.Second},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sem := New(tc.capacity, tc.timeout)
			ctx := context.Background()

			// Acquire all slots
			for i := 0; i < tc.capacity; i++ {
				if err := sem.Acquire(ctx); err != nil {
					t.Fatalf("Acquire() %d failed: %v", i, err)
				}
			}

			// Verify semaphore is exhausted
			if len(sem.sem) != 0 {
				t.Errorf("after acquiring all slots, len = %d; want 0", len(sem.sem))
			}

			// Release all slots
			for i := 0; i < tc.capacity; i++ {
				sem.Release()
			}

			// Verify all slots are available
			if len(sem.sem) != tc.capacity {
				t.Errorf("after releasing all slots, len = %d; want %d", len(sem.sem), tc.capacity)
			}
		})
	}
}

func TestAcquireTimeout(t *testing.T) {
	t.Parallel()

	sem := New(1, 100*time.Millisecond)
	ctx := context.Background()

	// Acquire the only slot
	if err := sem.Acquire(ctx); err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Try to acquire again - should timeout
	start := time.Now()
	err := sem.Acquire(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Acquire() should have timed out but succeeded")
	}

	if elapsed < 90*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("timeout took %v; want ~100ms", elapsed)
	}

	if err.Error() != "timeout acquiring connection slot after 100ms" {
		t.Errorf("error = %q; want timeout message", err)
	}
}

func TestAcquireContextCancellation(t *testing.T) {
	t.Parallel()

	sem := New(1, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	// Acquire the only slot
	if err := sem.Acquire(ctx); err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Cancel context immediately
	cancel()

	// Try to acquire with cancelled context
	err := sem.Acquire(ctx)
	if err != context.Canceled {
		t.Errorf("error = %v; want context.Canceled", err)
	}
}

func TestAcquireNilSemaphore(t *testing.T) {
	t.Parallel()

	var sem *ConnSemaphore
	ctx := context.Background()

	// Acquire should be no-op
	if err := sem.Acquire(ctx); err != nil {
		t.Errorf("Acquire() on nil semaphore failed: %v", err)
	}

	// Release should be no-op
	sem.Release()
}

func TestConcurrentAcquireRelease(t *testing.T) {
	t.Parallel()

	const (
		capacity   = 10
		goroutines = 100
		iterations = 50
	)

	sem := New(capacity, 1*time.Second)
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if err := sem.Acquire(ctx); err != nil {
					errors <- err
					return
				}
				// Simulate some work
				time.Sleep(time.Microsecond)
				sem.Release()
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}

	// Verify all slots were released
	if len(sem.sem) != capacity {
		t.Errorf("final len = %d; want %d", len(sem.sem), capacity)
	}
}

func TestAcquireReleasePattern(t *testing.T) {
	t.Parallel()

	sem := New(2, 1*time.Second)
	ctx := context.Background()

	// Acquire first slot
	if err := sem.Acquire(ctx); err != nil {
		t.Fatalf("first Acquire() failed: %v", err)
	}

	// Acquire second slot
	if err := sem.Acquire(ctx); err != nil {
		t.Fatalf("second Acquire() failed: %v", err)
	}

	// Both slots taken
	if len(sem.sem) != 0 {
		t.Errorf("after 2 acquires, len = %d; want 0", len(sem.sem))
	}

	// Release first
	sem.Release()
	if len(sem.sem) != 1 {
		t.Errorf("after 1 release, len = %d; want 1", len(sem.sem))
	}

	// Acquire again
	if err := sem.Acquire(ctx); err != nil {
		t.Fatalf("third Acquire() failed: %v", err)
	}

	// Release remaining
	sem.Release()
	sem.Release()
	if len(sem.sem) != 2 {
		t.Errorf("after all releases, len = %d; want 2", len(sem.sem))
	}
}

func TestAcquireWithDeadline(t *testing.T) {
	t.Parallel()

	sem := New(1, 5*time.Second)

	// Create context with tight deadline
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Acquire the only slot
	if err := sem.Acquire(ctx); err != nil {
		t.Fatalf("first Acquire() failed: %v", err)
	}

	// Try to acquire with deadline - should fail with context deadline
	start := time.Now()
	err := sem.Acquire(ctx)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("error = %v; want context.DeadlineExceeded", err)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("timeout took %v; should fail quickly with context deadline", elapsed)
	}
}
