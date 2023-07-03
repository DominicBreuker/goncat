package config

// Master ...
type Master struct {
	Exec    string
	Pty     bool
	LogFile string
}

// Validate ...
func (cfg *Master) Validate() []error {
	var errors []error

	return errors
}
