package format

import (
	"fmt"
	"strings"
)

func Addr(host string, port int) string {
	if strings.ContainsAny(host, ":") { // IPv6
		return fmt.Sprintf("[%s]:%d", host, port)
	} else { // IPv4
		return fmt.Sprintf("%s:%d", host, port)
	}
}
