package shared

import (
	"github.com/urfave/cli/v3"
)

const categoryCommon = "Common"

const SSLFlag = "ssl"
const KeyFlag = "key"
const ExecFlag = "exec"
const PtyFlag = "pty"
const LogFileFlag = "log"
const CleanupFlag = "cleanup"
const VerboseFlag = "verbose"

// GetFlags ...
func GetFlags() []cli.Flag {
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
		&cli.StringFlag{
			Name:     ExecFlag,
			Aliases:  []string{"e"},
			Usage:    "Execute program",
			Category: categoryCommon,
			Value:    "",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     PtyFlag,
			Aliases:  []string{},
			Usage:    "Enable Pty mode",
			Category: categoryCommon,
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     LogFileFlag,
			Aliases:  []string{"l"},
			Usage:    "Log file",
			Category: categoryCommon,
			Value:    "",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     CleanupFlag,
			Aliases:  []string{"c"},
			Usage:    "Clean up after running",
			Category: categoryCommon,
			Value:    false,
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
