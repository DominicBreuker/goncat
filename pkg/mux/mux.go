package mux

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

func OpenChannels(conn net.Conn) (net.Conn, net.Conn, error) {
	session, err := yamux.Client(conn, config())
	if err != nil {
		return nil, nil, fmt.Errorf("yamux.Client(conn): %s", err)
	}

	connCtl, err := session.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("session.Open(), ctl: %s", err)
	}

	connData, err := session.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("session.Open(), data: %s", err)
	}

	return connCtl, connData, nil
}

func AcceptChannels(conn net.Conn) (net.Conn, net.Conn, error) {
	session, err := yamux.Server(conn, config())
	if err != nil {
		return nil, nil, fmt.Errorf("yamux.Server(conn): %s", err)
	}

	connCtl, err := session.Accept()
	if err != nil {
		return nil, nil, fmt.Errorf("session.Accept(), ctl: %s", err)
	}

	connData, err := session.Accept()
	if err != nil {
		return nil, nil, fmt.Errorf("session.Accept(), data: %s", err)
	}

	return connCtl, connData, nil
}

func config() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = nil
	cfg.Logger = log.New(ioutil.Discard, "", log.LstdFlags) // discard all console logging in yamux
	return cfg
}
