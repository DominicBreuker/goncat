package config

import (
	"fmt"
	"strconv"
	"strings"
)

// LocalPortForwardingCfg ...
type LocalPortForwardingCfg portForwardingCfg

// RemotePortForwardingCfg ...
type RemotePortForwardingCfg portForwardingCfg

type portForwardingCfg struct {
	LocalHost  string
	LocalPort  int
	RemoteHost string
	RemotePort int

	spec       string
	parsingErr error
}

func (lpf *LocalPortForwardingCfg) String() string {
	if lpf.parsingErr != nil {
		return lpf.spec
	}

	return fmt.Sprintf("%s:%d:%s:%d", lpf.LocalHost, lpf.LocalPort, lpf.RemoteHost, lpf.RemotePort)
}

func (rpf *RemotePortForwardingCfg) String() string {
	if rpf.parsingErr != nil {
		return rpf.spec
	}

	return fmt.Sprintf("%s:%d:%s:%d", rpf.RemoteHost, rpf.RemotePort, rpf.LocalHost, rpf.LocalPort)
}

func newRemotePortForwardingCfg(spec string) *RemotePortForwardingCfg {
	lpf := newLocalPortForwardingCfg(spec)

	// Remote config is just local config but in reverse order
	return &RemotePortForwardingCfg{
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

	tokens := strings.Split(spec, ":")
	if len(tokens) != 3 && len(tokens) != 4 {
		out.parsingErr = fmt.Errorf("unexpected format")
		return &out
	}

	offset := 0

	if len(tokens) == 4 {
		offset++
		out.LocalHost = tokens[0]
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
