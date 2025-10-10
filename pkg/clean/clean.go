// Package clean provides functionality for ensuring executable deletion,
// useful for cleanup operations and self-deletion scenarios.
package clean

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// EnsureDeletion sets up signal handlers to delete the executable file on program termination.
// It returns a cleanup function that can be called explicitly to delete the executable,
// and an error if the executable path cannot be determined.
//
// The function registers handlers for SIGINT, SIGKILL, and os.Interrupt signals.
// When any of these signals are received, the executable file is deleted before the program exits.
//
// The returned cleanup function can be called explicitly (e.g., with defer) to ensure
// the executable is deleted even when the program exits normally.
func EnsureDeletion() (func(), error) {
	path, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("os.Executable(): %s", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGKILL, os.Interrupt)

	go func() {
		<-sigs
		deleteFile(path)
		os.Exit(0)
	}()

	return func() { deleteFile(path) }, nil
}
