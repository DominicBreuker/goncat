package slaveconnect

import (
	"testing"
)

func TestGetCommand(t *testing.T) {
	t.Parallel()

	cmd := GetCommand()

	if cmd == nil {
		t.Fatal("GetCommand() returned nil")
	}

	if cmd.Name != "connect" {
		t.Errorf("command name = %q; want %q", cmd.Name, "connect")
	}

	if cmd.Usage == "" {
		t.Error("command usage should not be empty")
	}

	if cmd.Action == nil {
		t.Error("command action should not be nil")
	}

	if cmd.Flags == nil {
		t.Error("command flags should not be nil")
	}
}

func TestGetFlags(t *testing.T) {
	t.Parallel()

	flags := getFlags()

	if flags == nil {
		t.Fatal("getFlags() returned nil")
	}

	if len(flags) == 0 {
		t.Error("getFlags() should return at least some flags")
	}

	// Verify common, slave, and connect flags are included
	flagNames := make(map[string]bool)
	for _, flag := range flags {
		if names := flag.Names(); len(names) > 0 {
			flagNames[names[0]] = true
		}
	}

	expectedFlags := []string{"ssl", "key", "verbose", "cleanup"}
	for _, name := range expectedFlags {
		if !flagNames[name] {
			t.Errorf("expected flag %q not found", name)
		}
	}
}
