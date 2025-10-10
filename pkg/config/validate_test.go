package config

import (
	"fmt"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfgs     []ValidatableConfig
		wantErrs int
	}{
		{
			name:     "no configs",
			cfgs:     []ValidatableConfig{},
			wantErrs: 0,
		},
		{
			name: "one valid config",
			cfgs: []ValidatableConfig{
				&Shared{Port: 8080},
			},
			wantErrs: 0,
		},
		{
			name: "one invalid config",
			cfgs: []ValidatableConfig{
				&Shared{Port: 0},
			},
			wantErrs: 1,
		},
		{
			name: "multiple configs with errors",
			cfgs: []ValidatableConfig{
				&Shared{Port: 0, SSL: false, Key: "key"},
				&Slave{},
			},
			wantErrs: 2,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := Validate(tc.cfgs...)
			if len(errs) != tc.wantErrs {
				t.Errorf("Validate() returned %d errors, want %d", len(errs), tc.wantErrs)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 1", 1, false},
		{"valid port 8080", 8080, false},
		{"valid port 65535", 65535, false},
		{"invalid port 0", 0, true},
		{"invalid port -1", -1, true},
		{"invalid port 65536", 65536, true},
		{"invalid port 100000", 100000, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePort(tc.port)
			if (err != nil) != tc.wantErr {
				t.Errorf("validatePort(%d) error = %v, wantErr %v", tc.port, err, tc.wantErr)
			}
		})
	}
}

// mockValidatableConfig is a mock implementation for testing.
type mockValidatableConfig struct {
	errors []error
}

func (m *mockValidatableConfig) Validate() []error {
	return m.errors
}

func TestValidate_Accumulates(t *testing.T) {
	t.Parallel()

	mock1 := &mockValidatableConfig{
		errors: []error{fmt.Errorf("error1"), fmt.Errorf("error2")},
	}
	mock2 := &mockValidatableConfig{
		errors: []error{fmt.Errorf("error3")},
	}

	errs := Validate(mock1, mock2)
	if len(errs) != 3 {
		t.Errorf("Validate() returned %d errors, want 3", len(errs))
	}
}
