// Package shared provides common CLI flag definitions and utility functions
// used across goncat's command-line interface.
package shared

import (
	"strings"

	"github.com/urfave/cli/v3"
)

const categoryCommon = "common"

// SSLFlag is the name of the flag to enable TLS encryption.
const SSLFlag = "ssl"

// KeyFlag is the name of the flag to specify the mTLS authentication key.
const KeyFlag = "key"

// VerboseFlag is the name of the flag to enable verbose error logging.
const VerboseFlag = "verbose"

// TimeoutFlag is the name of the flag to specify operation timeout in milliseconds.
const TimeoutFlag = "timeout" // TODO for future: consider changing to time.Duration type, cmd.Duration(...)

// GetBaseDescription returns the base description text for transport
// specifications used in CLI commands.
func GetBaseDescription() string {
	return strings.Join([]string{
		"Specify transport like this: tcp://127.0.0.1:123 (supports tcp|ws|wss|udp)",
		"You can omit the host when listening to bind to all interfaces.",
	}, "\n")
}

// GetArgsUsage returns the arguments usage string for CLI commands.
func GetArgsUsage() string {
	return strings.Join([]string{
		"transport",
	}, " ")
}

// GetCommonFlags returns the common CLI flags used by both master and slave modes.
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
		&cli.IntFlag{
			Name:     TimeoutFlag,
			Aliases:  []string{"t"},
			Usage:    "Operation timeout in milliseconds (TLS handshake, mux control operations)",
			Category: categoryCommon,
			Value:    10000, // 10 seconds default
			Required: false,
		},
	}
}

// GetConnectFlags returns the CLI flags specific to connect mode.
// Currently returns an empty slice.
func GetConnectFlags() []cli.Flag {
	return []cli.Flag{}
}

// GetListenFlags returns the CLI flags specific to listen mode.
// Currently returns an empty slice.
func GetListenFlags() []cli.Flag {
	return []cli.Flag{}
}

const categoryMaster = "master"

// ExecFlag is the name of the flag to specify a program to execute.
const ExecFlag = "exec"

// PtyFlag is the name of the flag to enable PTY mode.
const PtyFlag = "pty"

// LogFileFlag is the name of the flag to specify a log file.
const LogFileFlag = "log"

// LocalPortForwardingFlag is the name of the flag for local port forwarding.
const LocalPortForwardingFlag = "local"

// RemotePortForwardingFlag is the name of the flag for remote port forwarding.
const RemotePortForwardingFlag = "remote"

// SocksFlag is the name of the flag to enable SOCKS proxy mode.
const SocksFlag = "socks"

// GetMasterFlags returns the CLI flags specific to master mode.
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
			Usage:    "Local port forwarding: [[T:|U:]<local-host>:]<local-port>:<remote-host>:<remote-port>. Use T: for TCP (default), U: for UDP. Example: -L U:127.0.0.1:53:8.8.8.8:53",
			Category: categoryMaster,
			Value:    []string{},
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     RemotePortForwardingFlag,
			Aliases:  []string{"R"},
			Usage:    "Remote port forwarding: [[T:|U:]<remote-host>:]<remote-port>:<local-host>:<local-port>. Use T: for TCP (default), U: for UDP. Example: -R U:53:localhost:5353",
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

// CleanupFlag is the name of the flag to enable automatic cleanup after running.
const CleanupFlag = "cleanup"

// GetSlaveFlags returns the CLI flags specific to slave mode.
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
