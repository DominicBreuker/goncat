package version

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

var Version = "unknown"

func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Program version",
		Action: func(cCtx *cli.Context) error {
			fmt.Println(Version)
			return nil
		},
		Flags: []cli.Flag{},
	}
}
