package main

import (
	"dominicbreuker/goncat/cmd/master"
	"dominicbreuker/goncat/cmd/slave"
	"dominicbreuker/goncat/cmd/version"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"os"
	"runtime"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.App{
		Name:  "goncat",
		Usage: "netcat-like tool for reverse shells",
		Commands: []*cli.Command{
			master.GetCommand(),
			slave.GetCommand(),
			version.GetCommand(),
		},
	}

	// go func() {
	// 	for {
	// 		printRuntimeStats()
	// 		time.Sleep(1 * time.Second)
	// 	}
	// }()

	if err := app.Run(os.Args); err != nil {
		log.ErrorMsg("Run: %s\n", err)
	}
}

// printRuntimeStats is used only in development, sometimes...
func printRuntimeStats() {
	fmt.Printf("# Goroutiens = %d", runtime.NumGoroutine())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\tAlloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
