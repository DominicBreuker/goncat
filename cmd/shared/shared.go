package shared

import (
	"strings"

	"github.com/urfave/cli/v3"
)

const categoryCommon = "common"

const SSLFlag = "ssl"
const KeyFlag = "key"
const VerboseFlag = "verbose"

func GetBaseDescription() string {
	return strings.Join([]string{
		"Specify transport like this: tcp://127.0.0.1:123 (supports tcp|ws|wss)",
		"You can omit the host when listening to bind to all interfaces.",
	}, "\n")
}

func GetArgsUsage() string {
	return strings.Join([]string{
		"transport",
	}, " ")
}

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

// GetConnectFlags ...
func GetConnectFlags() []cli.Flag {
	return []cli.Flag{}
}

const categoryListen = "listen"

// GetListenFlags ...
func GetListenFlags() []cli.Flag {
	return []cli.Flag{}
}

const categoryMaster = "master"

const ExecFlag = "exec"
const PtyFlag = "pty"
const LogFileFlag = "log"
const LocalPortForwardingFlag = "local"
const RemotePortForwardingFlag = "remote"
const SocksFlag = "socks"

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
		&cli.StringFlag{
			Name:     SocksFlag,
			Aliases:  []string{"D"},
			Usage:    "SOCKS proxy, format: -D <local-host>:<local-port> (local-host optional and defaults to 127.0.0.1, specify :<local-port> to listen on all interfaces)",
			Category: categoryMaster,
			Value:    "",
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
