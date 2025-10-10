//go:build windows
// +build windows

package pty

import (
	"testing"
)

func TestTerminalSize_serialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		size     TerminalSize
		expected uintptr
	}{
		{
			name:     "standard 80x24",
			size:     TerminalSize{Rows: 24, Cols: 80},
			expected: uintptr((int32(24) << 16) | int32(80)),
		},
		{
			name:     "wide 120x40",
			size:     TerminalSize{Rows: 40, Cols: 120},
			expected: uintptr((int32(40) << 16) | int32(120)),
		},
		{
			name:     "zero values",
			size:     TerminalSize{Rows: 0, Cols: 0},
			expected: uintptr(0),
		},
		{
			name:     "max reasonable size",
			size:     TerminalSize{Rows: 200, Cols: 300},
			expected: uintptr((int32(200) << 16) | int32(300)),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.size.serialize()
			if got != tc.expected {
				t.Errorf("serialize() = 0x%x, want 0x%x", got, tc.expected)
			}
		})
	}
}

func TestCloseHandle_InvalidHandle(t *testing.T) {
	t.Parallel()

	// Closing an invalid handle should not error
	err := closeHandle(^uintptr(0)) // windows.InvalidHandle
	if err != nil {
		t.Errorf("closeHandle(InvalidHandle) error = %v, want nil", err)
	}
}
