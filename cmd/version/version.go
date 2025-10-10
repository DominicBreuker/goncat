// Package version provides the version command for goncat.
package version

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// Version is the version string of goncat, set at build time via ldflags.
var Version = "unknown"

// GetCommand returns the CLI command for displaying the version.
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Program version",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(Version)
			return nil
		},
		Flags: []cli.Flag{},
	}
}
