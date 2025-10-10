package config

// Slave contains configuration specific to slave mode operation.
type Slave struct {
	Clean bool
}

// Validate checks the Slave configuration for errors and returns any validation errors found.
func (cfg *Slave) Validate() []error {
	var errors []error

	return errors
}
