// Package clean provides functionality for ensuring executable deletion,
// useful for cleanup operations and self-deletion scenarios.
package clean

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"os"
)

// EnsureDeletion sets up the current executable to be deleted
func EnsureDeletion(ctx context.Context, logger *log.Logger) (func(), error) {
	path, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("os.Executable(): %s", err)
	}

	go func() {
		<-ctx.Done()
		deleteFile(path, logger)
	}()

	return func() { deleteFile(path, logger) }, nil
}
