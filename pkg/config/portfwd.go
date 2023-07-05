package config

import (
	"fmt"
	"strconv"
	"strings"
)

// LocalPortForwardingCfg ...
type LocalPortForwardingCfg struct {
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
		out.parsingErr = fmt.Errorf("parsing local port: %s", err)
		return &out
	}

	out.RemoteHost = tokens[1+offset]

	out.RemotePort, err = strconv.Atoi(tokens[2+offset])
	if err != nil {
		out.parsingErr = fmt.Errorf("parsing remote port: %s", err)
		return &out
	}

	return &out
}

func (lpf *LocalPortForwardingCfg) validate() []error {
	var errors []error

	if err := validatePort(lpf.LocalPort); err != nil {
		errors = append(errors, fmt.Errorf("local port: %s", err))
	}

	if len(lpf.RemoteHost) == 0 {
		errors = append(errors, fmt.Errorf("remote addr: must not be empty"))
	}

	if err := validatePort(lpf.RemotePort); err != nil {
		errors = append(errors, fmt.Errorf("remote port: %s", err))
	}

	return errors
}
