package shared

import (
	"strings"
	"testing"
)

func TestGetBaseDescription(t *testing.T) {
	t.Parallel()

	desc := GetBaseDescription()

	if desc == "" {
		t.Error("GetBaseDescription() should not return empty string")
	}

	if !strings.Contains(desc, "tcp") {
		t.Error("description should mention tcp protocol")
	}

	if !strings.Contains(desc, "ws") {
		t.Error("description should mention ws protocol")
	}

	if !strings.Contains(desc, "wss") {
		t.Error("description should mention wss protocol")
	}
}

func TestGetArgsUsage(t *testing.T) {
	t.Parallel()

	usage := GetArgsUsage()

	if usage == "" {
		t.Error("GetArgsUsage() should not return empty string")
	}

	if !strings.Contains(usage, "transport") {
		t.Error("usage should mention transport")
	}
}

func TestGetCommonFlags(t *testing.T) {
	t.Parallel()

	flags := GetCommonFlags()

	if flags == nil {
		t.Fatal("GetCommonFlags() returned nil")
	}

	if len(flags) == 0 {
		t.Error("GetCommonFlags() should return at least one flag")
	}

	// Check for expected flags
	flagNames := make(map[string]bool)
	for _, flag := range flags {
		if names := flag.Names(); len(names) > 0 {
			flagNames[names[0]] = true
		}
	}

	expectedFlags := []string{SSLFlag, KeyFlag, VerboseFlag}
	for _, name := range expectedFlags {
		if !flagNames[name] {
			t.Errorf("expected flag %q not found", name)
		}
	}
}

func TestGetConnectFlags(t *testing.T) {
	t.Parallel()

	flags := GetConnectFlags()

	if flags == nil {
		t.Fatal("GetConnectFlags() returned nil")
	}

	// Currently returns empty slice, but should not panic
}

func TestGetListenFlags(t *testing.T) {
	t.Parallel()

	flags := GetListenFlags()

	if flags == nil {
		t.Fatal("GetListenFlags() returned nil")
	}

	// Currently returns empty slice, but should not panic
}

func TestGetMasterFlags(t *testing.T) {
	t.Parallel()

	flags := GetMasterFlags()

	if flags == nil {
		t.Fatal("GetMasterFlags() returned nil")
	}

	if len(flags) == 0 {
		t.Error("GetMasterFlags() should return at least one flag")
	}

	// Check for expected flags
	flagNames := make(map[string]bool)
	for _, flag := range flags {
		if names := flag.Names(); len(names) > 0 {
			flagNames[names[0]] = true
		}
	}

	expectedFlags := []string{ExecFlag, PtyFlag, LogFileFlag, LocalPortForwardingFlag, RemotePortForwardingFlag, SocksFlag}
	for _, name := range expectedFlags {
		if !flagNames[name] {
			t.Errorf("expected flag %q not found", name)
		}
	}
}

func TestGetSlaveFlags(t *testing.T) {
	t.Parallel()

	flags := GetSlaveFlags()

	if flags == nil {
		t.Fatal("GetSlaveFlags() returned nil")
	}

	if len(flags) == 0 {
		t.Error("GetSlaveFlags() should return at least one flag")
	}

	// Check for cleanup flag
	flagNames := make(map[string]bool)
	for _, flag := range flags {
		if names := flag.Names(); len(names) > 0 {
			flagNames[names[0]] = true
		}
	}

	if !flagNames[CleanupFlag] {
		t.Errorf("expected flag %q not found", CleanupFlag)
	}
}

func TestFlagConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		constant string
	}{
		{"SSLFlag", SSLFlag},
		{"KeyFlag", KeyFlag},
		{"VerboseFlag", VerboseFlag},
		{"ExecFlag", ExecFlag},
		{"PtyFlag", PtyFlag},
		{"LogFileFlag", LogFileFlag},
		{"LocalPortForwardingFlag", LocalPortForwardingFlag},
		{"RemotePortForwardingFlag", RemotePortForwardingFlag},
		{"SocksFlag", SocksFlag},
		{"CleanupFlag", CleanupFlag},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.constant == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
		})
	}
}
