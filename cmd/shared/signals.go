package shared

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func SetupSignalHandling(cancel context.CancelFunc) {
	// ...existing code...
	sigCh := make(chan os.Signal, 2)

	// always handle Interrupt (portable)
	sigs := []os.Signal{os.Interrupt}

	// add Unix-only signals
	if runtime.GOOS != "windows" {
		sigs = append(sigs, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
		// SIGPIPE should generally be ignored to avoid process termination on broken pipes
		signal.Ignore(syscall.SIGPIPE)
	}

	signal.Notify(sigCh, sigs...)

	go func() {
		// first signal: request graceful shutdown
		s := <-sigCh
		cancel()

		// if a second signal arrives, force immediate exit.
		// otherwise allow a grace period for cleanup.
		select {
		case <-sigCh:
			// try to map to POSIX exit code 128+sig if possible
			if ss, ok := s.(syscall.Signal); ok {
				os.Exit(128 + int(ss))
			}
			os.Exit(1)
		case <-time.After(5 * time.Second):
			os.Exit(0)
		}
	}()
}
