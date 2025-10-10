package version

import (
	"context"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestGetCommand(t *testing.T) {
	t.Parallel()

	cmd := GetCommand()

	if cmd == nil {
		t.Fatal("GetCommand() returned nil")
	}

	if cmd.Name != "version" {
		t.Errorf("command name = %q; want %q", cmd.Name, "version")
	}

	if cmd.Usage == "" {
		t.Error("command usage should not be empty")
	}

	if cmd.Action == nil {
		t.Fatal("command action should not be nil")
	}
}

func TestVersionCommand_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
	}{
		{"default version", "unknown"},
		{"custom version", "1.2.3"},
		{"semver version", "v2.0.0-beta1"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Save and restore original version
			origVersion := Version
			defer func() { Version = origVersion }()

			Version = tt.version

			cmd := GetCommand()
			ctx := context.Background()
			cliCmd := &cli.Command{}

			// Execute the action
			err := cmd.Action(ctx, cliCmd)
			if err != nil {
				t.Errorf("Action() returned unexpected error: %v", err)
			}
		})
	}
}

func TestVersion_DefaultValue(t *testing.T) {
	t.Parallel()

	// Note: This test verifies the initial value but doesn't check if it's
	// changed at build time via ldflags.
	if Version == "" {
		t.Error("Version should have a default value")
	}
}
