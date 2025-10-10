// Package slave provides the slave command, which acts as the controlled
// side in a goncat connection. The slave can connect to or listen for masters.
package slave

import (
	"dominicbreuker/goncat/cmd/slaveconnect"
	"dominicbreuker/goncat/cmd/slavelisten"

	"github.com/urfave/cli/v3"
)

// GetCommand returns the CLI command for slave mode with its subcommands.
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "slave",
		Usage: "Act as a slave",
		Commands: []*cli.Command{
			slavelisten.GetCommand(),
			slaveconnect.GetCommand(),
		},
	}
}
