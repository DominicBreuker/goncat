package main

import (
	"dominicbreuker/goncat/cmd/master"
	"dominicbreuker/goncat/cmd/slave"
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
			master.GetCommand(),
			slave.GetCommand(),
			version.GetCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.ErrorMsg("Run: %s\n", err)
	}
}
