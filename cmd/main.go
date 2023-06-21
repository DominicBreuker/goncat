package main

import (
	"dominicbreuker/goncat/cmd/connect"
	"dominicbreuker/goncat/cmd/listen"
	"dominicbreuker/goncat/cmd/version"
	"dominicbreuker/goncat/pkg/log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.App{
		Name:  "goncat",
		Usage: "netcat-like tool for reverse shells",
		Commands: []*cli.Command{
			connect.GetCommand(),
			listen.GetCommand(),
			version.GetCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.ErrorMsg("Run: %s\n", err)
	}
}
