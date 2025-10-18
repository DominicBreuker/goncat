package mux

import (
	"context"
	"net"
	"testing"
	"time"
)

// Test that ReceiveContext returns promptly when the provided context is canceled
// even if the peer never sends a message. This ensures the goroutine used to
// interrupt blocking dec.Decode does not leak and cancellation is respected.
func TestReceiveContextCancellation_MasterAndSlave(t *testing.T) {
	// create an in-memory connected pair
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	// OpenSession/AcceptSession perform yamux handshake and open streams.
	// Run them concurrently to avoid timing issues on an in-memory pipe.
	type mres struct {
		s   *MasterSession
		err error
	}
	type sres struct {
		s   *SlaveSession
		err error
	}

	mch := make(chan mres, 1)
	sch := make(chan sres, 1)

	go func() {
		ms, err := OpenSessionContext(context.Background(), a, 50*time.Millisecond)
		mch <- mres{ms, err}
	}()

	go func() {
		ss, err := AcceptSessionContext(context.Background(), b, 50*time.Millisecond)
		sch <- sres{ss, err}
	}()

	var master *MasterSession
	var slave *SlaveSession

	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case rm := <-mch:
			if rm.err != nil {
				t.Fatalf("OpenSession(a): %v", rm.err)
			}
			master = rm.s
		case rs := <-sch:
			if rs.err != nil {
				t.Fatalf("AcceptSession(b): %v", rs.err)
			}
			slave = rs.s
		case <-timeout:
			t.Fatalf("timed out creating yamux sessions")
		}
	}

	defer master.Close()
	defer slave.Close()

	// Master.ReceiveContext should return when cancelled
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan error, 1)
		go func() {
			_, err := master.ReceiveContext(ctx)
			done <- err
		}()

		// give the goroutine a short moment to start and block
		time.Sleep(20 * time.Millisecond)

		start := time.Now()
		cancel()

		select {
		case err := <-done:
			if err == nil {
				t.Fatalf("expected non-nil error after cancel, got nil")
			}
			if time.Since(start) > 500*time.Millisecond {
				t.Fatalf("master.ReceiveContext did not return promptly after cancel: %v", time.Since(start))
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("master.ReceiveContext did not return after cancel within timeout")
		}
	}

	// Slave.ReceiveContext should also return when cancelled
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan error, 1)
		go func() {
			_, err := slave.ReceiveContext(ctx)
			done <- err
		}()

		// give the goroutine a short moment to start and block
		time.Sleep(20 * time.Millisecond)

		start := time.Now()
		cancel()

		select {
		case err := <-done:
			if err == nil {
				t.Fatalf("expected non-nil error after cancel, got nil")
			}
			if time.Since(start) > 500*time.Millisecond {
				t.Fatalf("slave.ReceiveContext did not return promptly after cancel: %v", time.Since(start))
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("slave.ReceiveContext did not return after cancel within timeout")
		}
	}
}
