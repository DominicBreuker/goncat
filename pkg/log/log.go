// Package log provides logging utilities including colored console output
// and connection logging capabilities.
package log

import (
	"os"
	"strings"

	"github.com/fatih/color"
)

var red = color.New(color.FgRed).FprintfFunc()
var blue = color.New(color.FgBlue).FprintfFunc()
var gray = color.New(color.FgHiBlack).FprintfFunc()

// Logger provides structured logging with verbose mode support.
type Logger struct {
	verbose bool
}

// NewLogger creates a new logger with the given verbose setting.
func NewLogger(verbose bool) *Logger {
	return &Logger{verbose: verbose}
}

// VerboseMsg logs a message only if verbose mode is enabled.
// It is safe to call on a nil Logger.
func (l *Logger) VerboseMsg(format string, a ...interface{}) {
	if l == nil || !l.verbose {
		return
	}
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	gray(os.Stderr, "[v] "+format, a...)
}

// ErrorMsg prints an error message to stderr in red color.
func (l *Logger) ErrorMsg(format string, a ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	red(os.Stderr, "[!] Error: "+format, a...)
}

// InfoMsg prints an informational message to stderr in blue color.
func (l *Logger) InfoMsg(format string, a ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	blue(os.Stderr, "[+] "+format, a...)
}

// Package-level default logger for backward compatibility
var defaultLogger = NewLogger(false)

// ErrorMsg prints an error message to stderr in red color.
// This is a package-level function for backward compatibility.
func ErrorMsg(format string, a ...interface{}) {
	defaultLogger.ErrorMsg(format, a...)
}

// InfoMsg prints an informational message to stderr in blue color.
// This is a package-level function for backward compatibility.
func InfoMsg(format string, a ...interface{}) {
	defaultLogger.InfoMsg(format, a...)
}
