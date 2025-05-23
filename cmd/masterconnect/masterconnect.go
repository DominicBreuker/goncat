package masterconnect

import (
	"context"
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
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

			if errors := config.Validate(cfg, mCfg); len(errors) > 0 {
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

			h, err := master.New(ctx, cfg, mCfg, c.GetConnection())
			if err != nil {
				return fmt.Errorf("master.New(): %s", err)
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
	flags = append(flags, shared.GetMasterFlags()...)
	flags = append(flags, shared.GetConnectFlags()...)

	return flags
}
