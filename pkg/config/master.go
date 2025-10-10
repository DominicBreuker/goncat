package config

import (
	"fmt"
)

// Master contains configuration specific to master mode operation,
// including command execution, PTY settings, logging, port forwarding, and SOCKS proxy.
type Master struct {
	Exec    string
	Pty     bool
	LogFile string

	LocalPortForwarding  []*LocalPortForwardingCfg
	RemotePortForwarding []*RemotePortForwardingCfg
	rpfDestinations      map[string]struct{}

	Socks *SocksCfg
}

func (mCfg *Master) getRpfDestinations() map[string]struct{} {
	if mCfg.rpfDestinations == nil {
		mCfg.rpfDestinations = make(map[string]struct{})
	}
	return mCfg.rpfDestinations
}

// ParseLocalPortForwardingSpecs parses local port forwarding specification strings
// and adds them to the master configuration.
func (mCfg *Master) ParseLocalPortForwardingSpecs(specs []string) {
	for _, spec := range specs {
		mCfg.addLocalPortForwardingSpec(spec)
	}
}

func (mCfg *Master) addLocalPortForwardingSpec(spec string) {
	mCfg.LocalPortForwarding = append(mCfg.LocalPortForwarding, newLocalPortForwardingCfg(spec))
}

// ParseRemotePortForwardingSpecs parses remote port forwarding specification strings
// and adds them to the master configuration.
func (mCfg *Master) ParseRemotePortForwardingSpecs(specs []string) {
	for _, spec := range specs {
		mCfg.addRemotePortForwardingSpec(spec)
	}
}

func (mCfg *Master) addRemotePortForwardingSpec(spec string) {
	rpf := newRemotePortForwardingCfg(spec)
	mCfg.RemotePortForwarding = append(mCfg.RemotePortForwarding, rpf)

	dests := mCfg.getRpfDestinations()
	dests[fmt.Sprintf("%s:%d", rpf.LocalHost, rpf.LocalPort)] = struct{}{}
}

// IsAllowedRemotePortForwardingDestination validates that remote port forwarding to the given host and port was actually requested
func (mCfg *Master) IsAllowedRemotePortForwardingDestination(host string, port int) bool {
	dests := mCfg.getRpfDestinations()
	_, ok := dests[fmt.Sprintf("%s:%d", host, port)]
	return ok
}

// IsSocksEnabled returns true if the SOCKS proxy feature is enabled
func (mCfg *Master) IsSocksEnabled() bool {
	return mCfg.Socks != nil
}

// Validate checks the Master configuration for errors and returns any validation errors found.
func (mCfg *Master) Validate() []error {
	var errors []error

	for _, lpf := range mCfg.LocalPortForwarding {
		if lpf.parsingErr != nil {
			errors = append(errors, fmt.Errorf("local port forwarding: %s: parsing error: %s", lpf, lpf.parsingErr))
			continue
		}

		for _, err := range lpf.validate() {
			errors = append(errors, fmt.Errorf("local port forwarding: %s: %s", lpf, err))
		}
	}

	for _, rpf := range mCfg.RemotePortForwarding {
		if rpf.parsingErr != nil {
			errors = append(errors, fmt.Errorf("remote port forwarding: %s: parsing error: %s", rpf, rpf.parsingErr))
			continue
		}

		for _, err := range rpf.validate() {
			errors = append(errors, fmt.Errorf("remote port forwarding: %s: %s", rpf, err))
		}
	}

	if mCfg.IsSocksEnabled() {
		if mCfg.Socks.parsingErr != nil {
			errors = append(errors, fmt.Errorf("socks: %s: parsing error: %s", mCfg.Socks, mCfg.Socks.parsingErr))
		} else {
			for _, err := range mCfg.Socks.validate() {
				errors = append(errors, fmt.Errorf("socks: %s", err))
			}
		}
	}

	return errors
}
