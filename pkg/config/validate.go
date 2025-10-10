package config

import "fmt"

// ValidatableConfig is an interface for configuration types that can be validated.
type ValidatableConfig interface {
	Validate() []error
}

// Validate validates multiple configuration objects and returns all validation errors.
func Validate(cfgs ...ValidatableConfig) []error {
	var out []error

	for _, cfg := range cfgs {
		out = append(out, cfg.Validate()...)
	}

	return out
}

// validatePort checks if a port number is in the valid range [1, 65535].
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%d not in [1, 65535]", port)
	}

	return nil
}
