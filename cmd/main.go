// Package main is the entry point for goncat, a netcat-like tool for creating
// bind or reverse shells with an SSH-like experience.
package main

import (
	"context"
	"dominicbreuker/goncat/cmd/master"
	"dominicbreuker/goncat/cmd/slave"
	"dominicbreuker/goncat/cmd/version"
	"dominicbreuker/goncat/pkg/log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:        "goncat",
		Description: "netcat-like tool for reverse shells",
		Commands: []*cli.Command{
			master.GetCommand(),
			slave.GetCommand(),
			version.GetCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		logger := log.NewLogger(false)
		logger.ErrorMsg("Run: %s\n", err)
	}
}
