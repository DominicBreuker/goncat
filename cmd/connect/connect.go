package connect

import (
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/clean"
	"dominicbreuker/goncat/pkg/client"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"fmt"

	"github.com/urfave/cli/v3"
)

const categoryConnect = "connect"

const hostFlag = "host"
const portFlag = "port"

// GetCommand ...
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "connect",
		Usage: "Connect to a remote host",
		Action: func(cCtx *cli.Context) error {
			if cCtx.Bool(shared.CleanupFlag) {
				delFunc, err := clean.EnsureDeletion()
				if err != nil {
					return fmt.Errorf("clean.EnsureDeletion(): %s", err)
				}
				defer delFunc()
			}

			cfg := config.Config{
				Host:    cCtx.String(hostFlag),
				Port:    cCtx.Int(portFlag),
				SSL:     cCtx.Bool(shared.SSLFlag),
				Key:     cCtx.String(shared.KeyFlag),
				Exec:    cCtx.String(shared.ExecFlag),
				Pty:     cCtx.Bool(shared.PtyFlag),
				LogFile: cCtx.String(shared.LogFileFlag),
				Verbose: cCtx.Bool(shared.VerboseFlag),
			}

			if errors := cfg.Validate(); len(errors) > 0 {
				log.ErrorMsg("Argument validation errors:\n")
				for _, err := range errors {
					log.ErrorMsg(" - %s\n", err)
				}
				return fmt.Errorf("exiting")
			}

			log.InfoMsg("Connecting to %s:%d\n", cfg.Host, cfg.Port)

			c := client.New(cfg)
			if err := c.Connect(); err != nil {
				return fmt.Errorf("connecting: %s", err)
			}

			return nil
		},
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:     hostFlag,
				Aliases:  []string{},
				Usage:    "Remote host (name or IP)",
				Category: categoryConnect,
				Required: true,
			},
			&cli.IntFlag{
				Name:     portFlag,
				Aliases:  []string{"p"},
				Usage:    "Remote port",
				Category: categoryConnect,
				Required: true,
			},
		}, shared.GetFlags()...),
	}
}
