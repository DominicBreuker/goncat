package masterlisten

import (
	"context"
	"dominicbreuker/goncat/cmd/shared"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/handler/master"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/server"
	"fmt"
	"net"
	"strings"

	"github.com/urfave/cli/v3"
)

// GetCommand ...
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "listen",
		Usage: "Listen for connections",
		Action: func(ctx context.Context, cmd *cli.Command) error {
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

			s, err := server.New(ctx, cfg, makeHandler(ctx, cfg, mCfg))
			if err != nil {
				return fmt.Errorf("server.New(): %s", err)
			}

			if err := s.Serve(); err != nil {
				return fmt.Errorf("serving: %s", err)
			}
			defer s.Close()

			return nil
		},
		Flags: getFlags(),
	}
}

func makeHandler(ctx context.Context, cfg *config.Shared, mCfg *config.Master) func(conn net.Conn) error {
	return func(conn net.Conn) error {
		defer log.InfoMsg("Connection to %s closed\n", conn.RemoteAddr())
		defer conn.Close()

		mst, err := master.New(ctx, cfg, mCfg, conn)
		if err != nil {
			return fmt.Errorf("master.New(): %s", err)
		}
		defer mst.Close()

		if err := mst.Handle(); err != nil {
			return fmt.Errorf("handle: %s", err)
		}

		return nil
	}
}

func getFlags() []cli.Flag {
	flags := []cli.Flag{}

	flags = append(flags, shared.GetCommonFlags()...)
	flags = append(flags, shared.GetMasterFlags()...)
	flags = append(flags, shared.GetListenFlags()...)

	return flags
}
