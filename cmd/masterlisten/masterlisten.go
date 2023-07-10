package masterlisten

import (
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
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

			if errors := config.Validate(cfg, mCfg); len(errors) > 0 {
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
			defer s.Close()

			for {
				conn, err := s.Accept()
				if err != nil {
					log.ErrorMsg("Accepting new connection: %s", err)
					continue
				}

				if err := handle(cfg, mCfg, conn); err != nil {
					log.ErrorMsg("Handling connection: %s\n", err)
					continue
				}
			}
		},
		Flags: getFlags(),
	}
}

func handle(cfg *config.Shared, mCfg *config.Master, conn net.Conn) error {
	defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
	defer conn.Close()

	mst, err := master.New(cfg, mCfg, conn)
	if err != nil {
		return fmt.Errorf("master.New(): %s", err)
	}
	defer mst.Close()

	if err := mst.Handle(); err != nil {
		return fmt.Errorf("handle: %s", err)
	}

	return nil
}

func getFlags() []cli.Flag {
	flags := []cli.Flag{}

	flags = append(flags, shared.GetCommonFlags()...)
	flags = append(flags, shared.GetMasterFlags()...)
	flags = append(flags, shared.GetListenFlags()...)

	return flags
}
