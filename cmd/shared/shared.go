package shared

import (
	"github.com/urfave/cli/v3"
)

const categoryCommon = "common"

const SSLFlag = "ssl"
const KeyFlag = "key"
const VerboseFlag = "verbose"

// GetFlags ...
func GetCommonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:     SSLFlag,
			Aliases:  []string{"s"},
			Usage:    "Use TLS encryption",
			Category: categoryCommon,
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     KeyFlag,
			Aliases:  []string{"k"},
			Usage:    "Key for mTLS authentication, leave empty to disable authentication",
			Category: categoryCommon,
			Value:    "",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     VerboseFlag,
			Aliases:  []string{"v"},
			Usage:    "Verbose error logging",
			Category: categoryCommon,
			Value:    false,
			Required: false,
		},
	}
}

const categoryConnect = "connect"

const HostFlag = "host"
const PortFlag = "port"

// GetConnectFlags ...
func GetConnectFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     HostFlag,
			Aliases:  []string{},
			Usage:    "Remote host (name or IP)",
			Category: categoryConnect,
			Required: true,
		},
		&cli.IntFlag{
			Name:     PortFlag,
			Aliases:  []string{"p"},
			Usage:    "Remote port",
			Category: categoryConnect,
			Required: true,
		},
	}
}

const categoryListen = "listen"

// GetListenFlags ...
func GetListenFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     HostFlag,
			Aliases:  []string{},
			Usage:    "Local interface, leave empty for all interfaces",
			Category: categoryListen,
			Value:    "",
			Required: false,
		},
		&cli.IntFlag{
			Name:     PortFlag,
			Aliases:  []string{"p"},
			Usage:    "Local port",
			Category: categoryListen,
			Required: true,
		},
	}
}

const categoryMaster = "master"

const ExecFlag = "exec"
const PtyFlag = "pty"
const LogFileFlag = "log"
const LocalPortForwardingFlag = "local"
const RemotePortForwardingFlag = "remote"

// GetMasterFlags ...
func GetMasterFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     ExecFlag,
			Aliases:  []string{"e"},
			Usage:    "Execute program",
			Category: categoryMaster,
			Value:    "",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     PtyFlag,
			Aliases:  []string{},
			Usage:    "Enable Pty mode",
			Category: categoryMaster,
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     LogFileFlag,
			Aliases:  []string{"l"},
			Usage:    "Log file",
			Category: categoryMaster,
			Value:    "",
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     LocalPortForwardingFlag,
			Aliases:  []string{"L"},
			Usage:    "Local port forwarding, format: -L <local-host>:<local-port>:<remote-host>:<rempote-port> (local-host optional)",
			Category: categoryMaster,
			Value:    []string{},
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     RemotePortForwardingFlag,
			Aliases:  []string{"R"},
			Usage:    "Remote port forwarding, format: -R <remote-host>:<remote-port>:<local-host>:<local-port> (remote-host optional)",
			Category: categoryMaster,
			Value:    []string{},
			Required: false,
		},
	}
}

const categorySlave = "slave"

const CleanupFlag = "cleanup"

func GetSlaveFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:     CleanupFlag,
			Aliases:  []string{"c"},
			Usage:    "Clean up after running",
			Category: categorySlave,
			Value:    false,
			Required: false,
		},
	}
}
