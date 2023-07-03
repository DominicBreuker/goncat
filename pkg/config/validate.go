package config

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
