// Package slavelisten implements the slave listen command, which listens
// for incoming master connections and follows their instructions.
package slavelisten

import (
	"context"
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/clean"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

// GetCommand returns the CLI command for slave listen mode.
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:        "listen",
		Description: shared.GetBaseDescription(),
		ArgsUsage:   shared.GetArgsUsage(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Bool(shared.CleanupFlag) {
				delFunc, err := clean.EnsureDeletion()
				if err != nil {
					return fmt.Errorf("clean.EnsureDeletion(): %s", err)
				}
				defer delFunc()
			}

			args := cmd.Args()
			if args.Len() != 1 {
				return fmt.Errorf("must provide exactly one argument, got %d (%s)", args.Len(), strings.Join(args.Slice(), ", "))
			}

			proto, host, port, err := shared.ParseTransport(args.Get(0))
			if err != nil {
				return fmt.Errorf("parsing transport: %s", err)
			}

			cfg := &config.Shared{
				ID:       fmt.Sprintf("slave[%s]", config.GenerateId()),
				Protocol: proto,
				Host:     host,
				Port:     port,
				SSL:      cmd.Bool(shared.SSLFlag),
				Key:      cmd.String(shared.KeyFlag),
				Verbose:  cmd.Bool(shared.VerboseFlag),
				Timeout:  time.Duration(cmd.Int(shared.TimeoutFlag)) * time.Millisecond,
			}

			if errs := config.Validate(cfg); len(errs) > 0 {
				log.ErrorMsg("Argument validation errors:")
				for _, err := range errs {
					log.ErrorMsg(" - %s", err)
				}
				return fmt.Errorf("exiting")
			}

			return entrypoint.SlaveListen(ctx, cfg)
		},
		Flags: getFlags(),
	}
}

func getFlags() []cli.Flag {
	flags := []cli.Flag{}

	flags = append(flags, shared.GetCommonFlags()...)
	flags = append(flags, shared.GetSlaveFlags()...)
	flags = append(flags, shared.GetListenFlags()...)

	return flags
}
