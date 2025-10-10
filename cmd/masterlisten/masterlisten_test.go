package masterlisten

import (
	"testing"
)

func TestGetCommand(t *testing.T) {
	t.Parallel()

	cmd := GetCommand()

	if cmd == nil {
		t.Fatal("GetCommand() returned nil")
	}

	if cmd.Name != "listen" {
		t.Errorf("command name = %q; want %q", cmd.Name, "listen")
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

	// Verify common, master, and listen flags are included
	flagNames := make(map[string]bool)
	for _, flag := range flags {
		if names := flag.Names(); len(names) > 0 {
			flagNames[names[0]] = true
		}
	}

	expectedFlags := []string{"ssl", "key", "verbose", "exec", "pty", "log"}
	for _, name := range expectedFlags {
		if !flagNames[name] {
			t.Errorf("expected flag %q not found", name)
		}
	}
}
