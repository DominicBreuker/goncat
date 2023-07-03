package slavelisten

import (
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/clean"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/slave"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/server"
	"fmt"
	"net"

	"github.com/urfave/cli/v3"
)

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

			cfg := &config.Shared{
				Host:    cCtx.String(shared.HostFlag),
				Port:    cCtx.Int(shared.PortFlag),
				SSL:     cCtx.Bool(shared.SSLFlag),
				Key:     cCtx.String(shared.KeyFlag),
				Verbose: cCtx.Bool(shared.VerboseFlag),
			}

			if errors := config.Validate(cfg); len(errors) > 0 {
				log.ErrorMsg("Argument validation errors:\n")
				for _, err := range errors {
					log.ErrorMsg(" - %s\n", err)
				}
				return fmt.Errorf("exiting")
			}

			s := server.New(cfg)
			if err := s.Serve(); err != nil {
				return fmt.Errorf("serving: %s", err)
			}

			for {
				conn, err := s.Accept()
				if err != nil {
					log.ErrorMsg("Accepting new connection: %s\n", err)
					continue
				}

				if err := handle(cfg, conn); err != nil {
					log.ErrorMsg("Handling connection: %s\n", err)
					continue
				}
			}
		},
		Flags: getFlags(),
	}
}

func handle(cfg *config.Shared, conn net.Conn) error {
	defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
	defer conn.Close()

	slv, err := slave.New(cfg, conn)
	if err != nil {
		return fmt.Errorf("slave.New(): %s", err)
	}
	defer slv.Close()

	if err := slv.Handle(); err != nil {
		return fmt.Errorf("handle: %s", err)
	}

	return nil
}

func getFlags() []cli.Flag {
	flags := []cli.Flag{}

	flags = append(flags, shared.GetCommonFlags()...)
	flags = append(flags, shared.GetSlaveFlags()...)
	flags = append(flags, shared.GetListenFlags()...)

	return flags
}
