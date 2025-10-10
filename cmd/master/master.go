// Package master provides the master command, which acts as the controlling
// side in a goncat connection. The master can connect to or listen for slaves.
package master

import (
	"dominicbreuker/goncat/cmd/masterconnect"
	"dominicbreuker/goncat/cmd/masterlisten"

	"github.com/urfave/cli/v3"
)

// GetCommand returns the CLI command for master mode with its subcommands.
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "master",
		Usage: "Act as a master",
		Commands: []*cli.Command{
			masterlisten.GetCommand(),
			masterconnect.GetCommand(),
		},
	}
}
