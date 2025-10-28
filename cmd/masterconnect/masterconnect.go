// Package masterconnect implements the master connect command, which connects
// to a remote slave and controls it.
package masterconnect

import (
	"context"
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/entrypoint"
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

// GetCommand returns the CLI command for master connect mode.
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:        "connect",
		Usage:       "Connect to a remote host",
		Description: shared.GetBaseDescription(),
		Action: func(parent context.Context, cmd *cli.Command) error {
			ctx, cancel := context.WithCancel(parent)
			defer cancel()

			shared.SetupSignalHandling(cancel)

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
				ID:       fmt.Sprintf("master[%s]", config.GenerateId()),
				Protocol: proto,
				Host:     host,
				Port:     port,
				SSL:      cmd.Bool(shared.SSLFlag),
				Key:      cmd.String(shared.KeyFlag),
				Verbose:  cmd.Bool(shared.VerboseFlag),
				Timeout:  time.Duration(cmd.Int(shared.TimeoutFlag)) * time.Millisecond,
				Logger:   log.NewLogger(cmd.Bool(shared.VerboseFlag)),
			}

			mCfg := &config.Master{
				Exec:    cmd.String(shared.ExecFlag),
				Pty:     cmd.Bool(shared.PtyFlag),
				LogFile: cmd.String(shared.LogFileFlag),
			}

			mCfg.ParseLocalPortForwardingSpecs(cmd.StringSlice(shared.LocalPortForwardingFlag))
			mCfg.ParseRemotePortForwardingSpecs(cmd.StringSlice(shared.RemotePortForwardingFlag))

			socksSpec := cmd.String(shared.SocksFlag)
			if socksSpec != "" {
				mCfg.Socks = config.NewSocksCfg(socksSpec)
			}

			if errs := config.Validate(cfg, mCfg); len(errs) > 0 {
				cfg.Logger.ErrorMsg("Argument validation errors:")
				for _, err := range errs {
					cfg.Logger.ErrorMsg(" - %s", err)
				}
				return fmt.Errorf("exiting")
			}

			return entrypoint.MasterConnect(ctx, cfg, mCfg)
		},
		Flags: getFlags(),
	}
}

func getFlags() []cli.Flag {
	flags := []cli.Flag{}

	flags = append(flags, shared.GetCommonFlags()...)
	flags = append(flags, shared.GetMasterFlags()...)
	flags = append(flags, shared.GetConnectFlags()...)

	return flags
}
