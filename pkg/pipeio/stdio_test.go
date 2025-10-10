package pipeio

import (
	"testing"
)

func TestNewStdio(t *testing.T) {
	t.Parallel()

	stdio := NewStdio()

	if stdio == nil {
		t.Fatal("NewStdio() returned nil")
	}
	if stdio.stdin == nil {
		t.Error("NewStdio() stdin is nil")
	}
	if stdio.stdout == nil {
		t.Error("NewStdio() stdout is nil")
	}
}

func TestStdio_Close(t *testing.T) {
	t.Parallel()

	stdio := NewStdio()

	if err := stdio.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
