// Package log provides logging utilities including colored console output
// and connection logging capabilities.
package log

import (
	"os"

	"github.com/fatih/color"
)

var red = color.New(color.FgRed).FprintfFunc()
var blue = color.New(color.FgBlue).FprintfFunc()

// ErrorMsg prints an error message to stderr in red color.
func ErrorMsg(format string, a ...interface{}) {
	red(os.Stderr, "[!] Error: "+format, a...)
}

// InfoMsg prints an informational message to stderr in blue color.
func InfoMsg(format string, a ...interface{}) {
	blue(os.Stderr, "[+] "+format, a...)
}
