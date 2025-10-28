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
		Action: func(parent context.Context, cmd *cli.Command) error {
			ctx, cancel := context.WithCancel(parent)
			defer cancel()

			shared.SetupSignalHandling(cancel)

			// Initialize logger early for cleanup logging
			logger := log.NewLogger(cmd.Bool(shared.VerboseFlag))

			if cmd.Bool(shared.CleanupFlag) {
				logger.VerboseMsg("Cleanup enabled: executable will be deleted on exit")
				delFunc, err := clean.EnsureDeletion(ctx, logger)
				if err != nil {
					return fmt.Errorf("clean.EnsureDeletion(): %s", err)
				}
				defer func() {
					logger.VerboseMsg("Executing cleanup: deleting executable")
					delFunc()
				}()
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
				Logger:   logger,
			}

			if errs := config.Validate(cfg); len(errs) > 0 {
				cfg.Logger.ErrorMsg("Argument validation errors:")
				for _, err := range errs {
					cfg.Logger.ErrorMsg(" - %s", err)
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
