package main

import (
	"dominicbreuker/goncat/cmd/connect"
	"dominicbreuker/goncat/cmd/listen"
	"dominicbreuker/goncat/cmd/version"
	"fmt"
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
		fmt.Printf("[!] Error: %s\n", err)
	}
}
