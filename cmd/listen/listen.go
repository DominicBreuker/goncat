package listen

import (
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/clean"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/server"
	"fmt"

	"github.com/urfave/cli/v3"
)

const categoryConnect = "connect"

const hostFlag = "host"
const portFlag = "port"

// GetCommand ...
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "listen",
		Usage: "Listen for connections",
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

			log.InfoMsg("Listening on %s:%d\n", cfg.Host, cfg.Port)

			s := server.New(cfg)
			if err := s.Serve(); err != nil {
				return fmt.Errorf("serving: %s", err)
			}

			return nil
		},
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:     hostFlag,
				Aliases:  []string{},
				Usage:    "Local interface, leave empty for all interfaces",
				Category: categoryConnect,
				Value:    "",
				Required: false,
			},
			&cli.IntFlag{
				Name:     portFlag,
				Aliases:  []string{"p"},
				Usage:    "Local port",
				Category: categoryConnect,
				Required: true,
			},
		}, shared.GetFlags()...),
	}
}
