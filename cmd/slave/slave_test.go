package slave

import (
	"testing"
)

func TestGetCommand(t *testing.T) {
	t.Parallel()

	cmd := GetCommand()

	if cmd == nil {
		t.Fatal("GetCommand() returned nil")
	}

	if cmd.Name != "slave" {
		t.Errorf("command name = %q; want %q", cmd.Name, "slave")
	}

	if cmd.Usage == "" {
		t.Error("command usage should not be empty")
	}

	if len(cmd.Commands) == 0 {
		t.Error("slave command should have subcommands")
	}

	// Verify subcommands exist
	expectedSubcommands := map[string]bool{
		"listen":  false,
		"connect": false,
	}

	for _, subcmd := range cmd.Commands {
		if _, ok := expectedSubcommands[subcmd.Name]; ok {
			expectedSubcommands[subcmd.Name] = true
		}
	}

	for name, found := range expectedSubcommands {
		if !found {
			t.Errorf("missing expected subcommand: %q", name)
		}
	}
}
