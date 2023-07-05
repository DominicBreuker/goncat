package config

import "fmt"

// ValidatableConfig ...
type ValidatableConfig interface {
	Validate() []error
}

// Validate ...
func Validate(cfgs ...ValidatableConfig) []error {
	var out []error

	for _, cfg := range cfgs {
		out = append(out, cfg.Validate()...)
	}

	return out
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%d not in [1, 65535]", port)
	}

	return nil
}
