package clean

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

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
