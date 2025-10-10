package config

import (
	"testing"
)

func TestSlave_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *Slave
	}{
		{
			name: "clean enabled",
			cfg:  &Slave{Clean: true},
		},
		{
			name: "clean disabled",
			cfg:  &Slave{Clean: false},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := tc.cfg.Validate()
			if len(errs) != 0 {
				t.Errorf("Slave.Validate() returned %d errors, want 0", len(errs))
			}
		})
	}
}
