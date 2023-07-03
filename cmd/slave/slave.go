package slave

import (
	"dominicbreuker/goncat/cmd/slaveconnect"
	"dominicbreuker/goncat/cmd/slavelisten"

	"github.com/urfave/cli/v3"
)

// GetCommand ...
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
