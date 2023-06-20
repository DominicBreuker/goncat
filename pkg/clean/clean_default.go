//go:build !windows
// +build !windows

package clean

import (
	"dominicbreuker/goncat/pkg/log"
	"os"
)

func deleteFile(path string) {
	if err := os.Remove(path); err != nil {
		log.ErrorMsg("deleting executable at %s: %s", path, err)
	}
}
