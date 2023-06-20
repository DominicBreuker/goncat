package log

import (
	"os"

	"github.com/fatih/color"
)

var red = color.New(color.FgRed).FprintfFunc()
var blue = color.New(color.FgBlue).FprintfFunc()

func ErrorMsg(format string, a ...interface{}) {
	red(os.Stderr, "[!] Error: "+format, a...)
}

func InfoMsg(format string, a ...interface{}) {
	blue(os.Stderr, "[+] "+format, a...)
}
