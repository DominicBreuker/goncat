package config

import (
	"fmt"
	"strconv"
	"strings"
)

// LocalPortForwardingCfg contains configuration for local port forwarding.
// Format: [localHost:]localPort:remoteHost:remotePort
type LocalPortForwardingCfg portForwardingCfg

// RemotePortForwardingCfg contains configuration for remote port forwarding.
// Format: [remoteHost:]remotePort:localHost:localPort
type RemotePortForwardingCfg portForwardingCfg

// portForwardingCfg is the underlying structure for both local and remote port forwarding.
type portForwardingCfg struct {
	Protocol   string // "tcp" or "udp", defaults to "tcp"
	LocalHost  string
	LocalPort  int
	RemoteHost string
	RemotePort int

	spec       string
	parsingErr error
}

// String returns the string representation of the local port forwarding configuration.
func (lpf *LocalPortForwardingCfg) String() string {
	if lpf.parsingErr != nil {
		return lpf.spec
	}

	proto := ""
	if lpf.Protocol != "tcp" {
		// Output single letter U for udp
		proto = "U:"
	}

	if lpf.LocalHost != "" {
		return fmt.Sprintf("%s%s:%d:%s:%d", proto, lpf.LocalHost, lpf.LocalPort, lpf.RemoteHost, lpf.RemotePort)
	}
	return fmt.Sprintf("%s%d:%s:%d", proto, lpf.LocalPort, lpf.RemoteHost, lpf.RemotePort)
}

// String returns the string representation of the remote port forwarding configuration.
func (rpf *RemotePortForwardingCfg) String() string {
	if rpf.parsingErr != nil {
		return rpf.spec
	}

	proto := ""
	if rpf.Protocol != "tcp" {
		// Output single letter U for udp
		proto = "U:"
	}

	if rpf.RemoteHost != "" {
		return fmt.Sprintf("%s%s:%d:%s:%d", proto, rpf.RemoteHost, rpf.RemotePort, rpf.LocalHost, rpf.LocalPort)
	}
	return fmt.Sprintf("%s%d:%s:%d", proto, rpf.RemotePort, rpf.LocalHost, rpf.LocalPort)
}

func newRemotePortForwardingCfg(spec string) *RemotePortForwardingCfg {
	lpf := newLocalPortForwardingCfg(spec)

	// Remote config is just local config but in reverse order
	return &RemotePortForwardingCfg{
		Protocol:   lpf.Protocol,
		LocalHost:  lpf.RemoteHost,
		LocalPort:  lpf.RemotePort,
		RemoteHost: lpf.LocalHost,
		RemotePort: lpf.LocalPort,

		spec:       lpf.spec,
		parsingErr: lpf.parsingErr,
	}
}

func newLocalPortForwardingCfg(spec string) *LocalPortForwardingCfg {
	var out LocalPortForwardingCfg
	out.spec = spec
	out.Protocol = "tcp" // Default protocol

	// Parse optional protocol prefix (T: or U: or t: or u:)
	tokens := strings.Split(spec, ":")
	if len(tokens) < 3 {
		out.parsingErr = fmt.Errorf("unexpected format")
		return &out
	}

	offset := 0

	// Check if first token is a protocol prefix (single letter T or U)
	if len(tokens[0]) <= 2 && (strings.ToUpper(tokens[0]) == "T" || strings.ToUpper(tokens[0]) == "U") {
		protocolMap := map[string]string{"T": "tcp", "U": "udp"}
		out.Protocol = protocolMap[strings.ToUpper(tokens[0])]
		offset = 1
	}

	// Now we need 3 or 4 remaining tokens (after protocol prefix if present)
	remainingTokens := len(tokens) - offset
	if remainingTokens != 3 && remainingTokens != 4 {
		out.parsingErr = fmt.Errorf("unexpected format")
		return &out
	}

	// Check if we have a local host (4 remaining tokens)
	if remainingTokens == 4 {
		out.LocalHost = tokens[offset]
		offset++
	}

	var err error

	out.LocalPort, err = strconv.Atoi(tokens[offset])
	if err != nil {
		out.parsingErr = fmt.Errorf("parsing '%s' as port: %s", tokens[offset], err)
		return &out
	}

	out.RemoteHost = tokens[1+offset]

	out.RemotePort, err = strconv.Atoi(tokens[2+offset])
	if err != nil {
		out.parsingErr = fmt.Errorf("parsing '%s' as port: %s", tokens[2+offset], err)
		return &out
	}

	return &out
}

func (rpf *RemotePortForwardingCfg) validate() []error {
	var errors []error

	errors = append(errors, portForwardingCfg(*rpf).validatePorts()...)

	if len(rpf.LocalHost) == 0 {
		errors = append(errors, fmt.Errorf("local host: must not be empty"))
	}

	return errors
}

func (lpf *LocalPortForwardingCfg) validate() []error {
	var errors []error

	errors = append(errors, portForwardingCfg(*lpf).validatePorts()...)

	if len(lpf.RemoteHost) == 0 {
		errors = append(errors, fmt.Errorf("remote host: must not be empty"))
	}

	return errors
}

func (pfw portForwardingCfg) validatePorts() []error {
	var errors []error

	if err := validatePort(pfw.LocalPort); err != nil {
		errors = append(errors, fmt.Errorf("local port: %s", err))
	}

	if err := validatePort(pfw.RemotePort); err != nil {
		errors = append(errors, fmt.Errorf("remote port: %s", err))
	}

	return errors
}
