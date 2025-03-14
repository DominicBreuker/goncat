package slaveconnect

import (
	"context"
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/clean"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

// GetCommand ...
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "connect",
		Usage: "Connect to a remote host",
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
			if host == "" {
				return fmt.Errorf("parsing transport: %s: specify a host", args.Get(0))
			}

			cfg := &config.Shared{
				Protocol: proto,
				Host:     host,
				Port:     port,
				SSL:      cmd.Bool(shared.SSLFlag),
				Key:      cmd.String(shared.KeyFlag),
				Verbose:  cmd.Bool(shared.VerboseFlag),
			}

			if errors := config.Validate(cfg); len(errors) > 0 {
				log.ErrorMsg("Argument validation errors:\n")
				for _, err := range errors {
					log.ErrorMsg(" - %s\n", err)
				}
				return fmt.Errorf("exiting")
			}

			c := client.New(ctx, cfg)
			if err := c.Connect(); err != nil {
				return fmt.Errorf("connecting: %s", err)
			}
			defer c.Close()

			h, err := slave.New(ctx, cfg, c.GetConnection())
			if err != nil {
				return fmt.Errorf("slave.New(): %s", err)
			}
			defer h.Close()

			if err := h.Handle(); err != nil {
				return fmt.Errorf("handling: %s", err)
			}

			return nil
		},
		Flags: getFlags(),
	}
}

func getFlags() []cli.Flag {
	flags := []cli.Flag{}

	flags = append(flags, shared.GetCommonFlags()...)
	flags = append(flags, shared.GetSlaveFlags()...)
	flags = append(flags, shared.GetConnectFlags()...)

	return flags
}
