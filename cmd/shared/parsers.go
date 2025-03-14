package shared

import (
	"dominicbreuker/goncat/pkg/config"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

func ParseTransport(s string) (proto config.Protocol, host string, port int, err error) {
	re := regexp.MustCompile(`^(tcp|ws|wss)://([^:]*):(\d+)$`)
	matches := re.FindStringSubmatch(s)

	if len(matches) != 4 {
		err = parsingError(s)
		return
	}

	switch matches[1] {
	case "tcp":
		proto = config.ProtoTCP
	case "ws":
		proto = config.ProtoWS
	case "wss":
		proto = config.ProtoWSS
	default:
		err = parsingError(s)
		return
	}
	host = matches[2]
	if host == "*" { // also counts as all interfaces
		host = ""
	}

	port, err = strconv.Atoi(matches[3])
	if err != nil || port < 1 || port > 65535 {
		err = parsingError(s)
		return
	}

	return
}

func parsingError(s string) error {
	return errors.New(fmt.Sprintf("parsing %s: format should be 'protocol://host:port', where protocol = tcp|ws|wss", s))
}
