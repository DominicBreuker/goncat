package slave

import (
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"fmt"
	"net"
)

func (slv *Slave) handleConnectAsync(m msg.Connect) {
	go func() {
		if err := slv.handleConnect(m); err != nil {
			log.ErrorMsg("Running connect job: %s", err)
		}
	}()
}

func (slv *Slave) handleConnect(m msg.Connect) error {
	connRemote, err := slv.sess.AcceptNewChannel()
	if err != nil {
		return fmt.Errorf("AcceptNewChannel(): %s", err)
	}
	defer connRemote.Close()

	addr := fmt.Sprintf("%s:%d", m.RemoteHost, m.RemotePort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	connLocal, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return fmt.Errorf("net.Dial(tcp, %s): %s", addr, err)
	}
	defer connLocal.Close()

	connLocal.SetKeepAlive(true)

	pipeio.Pipe(connRemote, connLocal, func(err error) {
		if slv.cfg.Verbose {
			log.ErrorMsg("Handling connect to %s: %s", addr, err)
		}
	})

	return nil
}
