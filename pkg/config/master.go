package config

import (
	"fmt"
)

// Master ...
type Master struct {
	Exec    string
	Pty     bool
	LogFile string

	LocalPortForwarding  []*LocalPortForwardingCfg
	RemotePortForwarding []*RemotePortForwardingCfg
}

// ParseLocalPortForwardingSpecs ...
func (mCfg *Master) ParseLocalPortForwardingSpecs(specs []string) {
	for _, spec := range specs {
		mCfg.addLocalPortForwardingSpec(spec)
	}
}

func (mCfg *Master) addLocalPortForwardingSpec(spec string) {
	mCfg.LocalPortForwarding = append(mCfg.LocalPortForwarding, newLocalPortForwardingCfg(spec))
}

// ParseRemotePortForwardingSpecs ...
func (mCfg *Master) ParseRemotePortForwardingSpecs(specs []string) {
	for _, spec := range specs {
		mCfg.addRemotePortForwardingSpec(spec)
	}
}

func (mCfg *Master) addRemotePortForwardingSpec(spec string) {
	mCfg.RemotePortForwarding = append(mCfg.RemotePortForwarding, newRemotePortForwardingCfg(spec))
}

// Validate ...
func (mCfg *Master) Validate() []error {
	var errors []error

	for _, lpf := range mCfg.LocalPortForwarding {
		if lpf.parsingErr != nil {
			errors = append(errors, fmt.Errorf("Local port forwarding: %s: parsing error: %s", lpf, lpf.parsingErr))
			continue
		}

		for _, err := range lpf.validate() {
			errors = append(errors, fmt.Errorf("Local port forwarding: %s: %s", lpf, err))
		}
	}

	for _, rpf := range mCfg.RemotePortForwarding {
		if rpf.parsingErr != nil {
			errors = append(errors, fmt.Errorf("Remote port forwarding: %s: parsing error: %s", rpf, rpf.parsingErr))
			continue
		}

		for _, err := range rpf.validate() {
			errors = append(errors, fmt.Errorf("Remote port forwarding: %s: %s", rpf, err))
		}
	}

	return errors
}
