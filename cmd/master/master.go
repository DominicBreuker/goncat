package master

import (
	"dominicbreuker/goncat/cmd/masterconnect"
	"dominicbreuker/goncat/cmd/masterlisten"

	"github.com/urfave/cli/v3"
)

// GetCommand ...
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
