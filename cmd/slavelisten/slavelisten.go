// Package slavelisten implements the slave listen command, which listens
// for incoming master connections and follows their instructions.
package slavelisten

import (
	"context"
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/clean"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/server"
	"fmt"
	"net"
	"strings"

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

			s, err := server.New(ctx, cfg, makeHandler(ctx, cfg))
			if err != nil {
				return fmt.Errorf("server.New(): %s", err)
			}

			if err := s.Serve(); err != nil {
				return fmt.Errorf("serving: %s", err)
			}

			return nil
		},
		Flags: getFlags(),
	}
}

func makeHandler(ctx context.Context, cfg *config.Shared) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
		defer conn.Close()

		slv, err := slave.New(ctx, cfg, conn)
		if err != nil {
			return fmt.Errorf("slave.New(): %s", err)
		}
		defer slv.Close()

		if err := slv.Handle(); err != nil {
			return fmt.Errorf("handle: %s", err)
		}

		return nil
	}
}

func getFlags() []cli.Flag {
	flags := []cli.Flag{}

	flags = append(flags, shared.GetCommonFlags()...)
	flags = append(flags, shared.GetSlaveFlags()...)
	flags = append(flags, shared.GetListenFlags()...)

	return flags
}
