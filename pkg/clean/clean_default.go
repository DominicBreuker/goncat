//go:build !windows
// +build !windows

package clean

import (
	"dominicbreuker/goncat/pkg/log"
	"os"
)

// deleteFile removes the file at the specified path.
// On non-Windows systems, this is a simple file removal operation.
// Errors are logged but not returned.
func deleteFile(path string, logger *log.Logger) {
	if err := os.Remove(path); err != nil {
		logger.ErrorMsg("deleting executable at %s: %s", path, err)
	}
}
