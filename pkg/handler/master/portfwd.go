package master

import (
	"context"
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"dominicbreuker/goncat/pkg/pipeio"
	"fmt"
	"net"
	"sync"
)

func (mst *Master) startLocalPortFwdJobJob(ctx context.Context, wg *sync.WaitGroup, lpf *config.LocalPortForwardingCfg) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := mst.handleLocalPortForwarding(ctx, lpf); err != nil {
			log.ErrorMsg("Local port forwarding: %s: %s\n", lpf, err)
		}
	}()
}

func (mst *Master) handleLocalPortForwarding(ctx context.Context, lpf *config.LocalPortForwardingCfg) error {
	addr := fmt.Sprintf("%s:%d", lpf.LocalHost, lpf.LocalPort)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.ResolveTCPAddr(tcp, %s): %s", addr, err)
	}

	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("listen(tcp, %s): %s", addr, err)
	}

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil // cancelled
			}

			log.ErrorMsg("Local port forwarding %s: Accept(): %s\n", lpf, err)
			continue
		}

		go func() {
			defer conn.Close()

			if err := mst.handleLocalPortForwardingConn(lpf, conn); err != nil {
				log.ErrorMsg("Local port forwarding %s: handling connection: %s", lpf, err)
			}
		}()
	}
}

func (mst *Master) handleLocalPortForwardingConn(lpf *config.LocalPortForwardingCfg, connLocal net.Conn) error {
	m := msg.Connect{
		RemoteHost: lpf.RemoteHost,
		RemotePort: lpf.RemotePort,
	}

	connRemote, err := mst.sess.SendAndOpenOneChannel(m)
	if err != nil {
		return fmt.Errorf("SendAndOpenOneChannel() for conn: %s", err)
	}
	defer connRemote.Close()

	pipeio.Pipe(connLocal, connRemote, func(err error) {
		if mst.cfg.Verbose {
			log.ErrorMsg("Pipe(stdio, conn): %s\n", err)
		}
	})

	return nil
}
