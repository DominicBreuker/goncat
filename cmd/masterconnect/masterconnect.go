package masterconnect

import (
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"dominicbreuker/goncat/pkg/log"
	"fmt"

	"github.com/urfave/cli/v3"
)

// GetCommand ...
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "connect",
		Usage: "Connect to a remote host",
		Action: func(cCtx *cli.Context) error {
			cfg := &config.Shared{
				Host:    cCtx.String(shared.HostFlag),
				Port:    cCtx.Int(shared.PortFlag),
				SSL:     cCtx.Bool(shared.SSLFlag),
				Key:     cCtx.String(shared.KeyFlag),
				Verbose: cCtx.Bool(shared.VerboseFlag),
			}

			mCfg := &config.Master{
				Exec:    cCtx.String(shared.ExecFlag),
				Pty:     cCtx.Bool(shared.PtyFlag),
				LogFile: cCtx.String(shared.LogFileFlag),
			}

			mCfg.ParseLocalPortForwardingSpecs(cCtx.StringSlice(shared.LocalPortForwardingFlag))
			mCfg.ParseRemotePortForwardingSpecs(cCtx.StringSlice(shared.RemotePortForwardingFlag))

			socksSpec := cCtx.String(shared.SocksFlag)
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

			c := client.New(cfg)
			if err := c.Connect(); err != nil {
				return fmt.Errorf("connecting: %s", err)
			}
			defer c.Close()

			h, err := master.New(cfg, mCfg, c.GetConnection())
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
