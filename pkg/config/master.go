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
	rpfDestinations      map[string]struct{}

	Socks *SocksCfg
}

func (mCfg *Master) getRpfDestinations() map[string]struct{} {
	if mCfg.rpfDestinations == nil {
		mCfg.rpfDestinations = make(map[string]struct{})
	}
	return mCfg.rpfDestinations
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

	if mCfg.IsSocksEnabled() {
		if mCfg.Socks.parsingErr != nil {
			errors = append(errors, fmt.Errorf("Socks: %s: parsing error: %s", mCfg.Socks, mCfg.Socks.parsingErr))
		} else {
			for _, err := range mCfg.Socks.validate() {
				errors = append(errors, fmt.Errorf("SOCKS: %s", err))
			}
		}
	}

	return errors
}
