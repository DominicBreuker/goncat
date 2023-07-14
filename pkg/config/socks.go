package config

import (
	"fmt"
	"strconv"
	"strings"
)

type SocksCfg struct {
	Host string
	Port int

	spec       string
	parsingErr error
}

func (sCfg *SocksCfg) String() string {
	if sCfg.parsingErr != nil {
		return sCfg.spec
	}

	return fmt.Sprintf("%s:%d", sCfg.Host, sCfg.Port)
}

func NewSocksCfg(spec string) *SocksCfg {
	var out SocksCfg
	out.spec = spec

	tokens := strings.Split(spec, ":")
	if len(tokens) != 1 && len(tokens) != 2 {
		out.parsingErr = fmt.Errorf("unexpected format")
		return &out
	}

	offset := 0

	if len(tokens) == 2 {
		offset++
		out.Host = tokens[0]
	} else {
		out.Host = "127.0.0.1"
	}

	var err error

	out.Port, err = strconv.Atoi(tokens[offset])
	if err != nil {
		out.parsingErr = fmt.Errorf("parsing '%s' as port: %s", tokens[offset], err)
		return &out
	}

	return &out
}

func (sCfg *SocksCfg) validate() []error {
	var errors []error

	if err := validatePort(sCfg.Port); err != nil {
		errors = append(errors, fmt.Errorf("port: %s", err))
	}

	return errors
}
