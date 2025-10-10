// Package pty provides cross-platform pseudo-terminal (PTY) functionality
// for creating and managing terminal sessions. It supports Unix systems
// (Linux, Darwin) via standard PTY operations and Windows via ConPTY.
package pty

// TerminalSize represents the dimensions of a terminal window in rows and columns.
type TerminalSize struct {
	Rows int // Number of rows (height) in the terminal
	Cols int // Number of columns (width) in the terminal
}
