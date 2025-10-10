package pty

import (
	"testing"
)

func TestTerminalSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size TerminalSize
		rows int
		cols int
	}{
		{
			name: "standard terminal",
			size: TerminalSize{Rows: 24, Cols: 80},
			rows: 24,
			cols: 80,
		},
		{
			name: "wide terminal",
			size: TerminalSize{Rows: 40, Cols: 120},
			rows: 40,
			cols: 120,
		},
		{
			name: "zero values",
			size: TerminalSize{Rows: 0, Cols: 0},
			rows: 0,
			cols: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.size.Rows != tc.rows {
				t.Errorf("Rows = %d, want %d", tc.size.Rows, tc.rows)
			}
			if tc.size.Cols != tc.cols {
				t.Errorf("Cols = %d, want %d", tc.size.Cols, tc.cols)
			}
		})
	}
}
