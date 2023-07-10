package slave

import (
	"dominicbreuker/goncat/pkg/config"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux"
	"dominicbreuker/goncat/pkg/mux/msg"
	"fmt"
	"io"
	"net"
)

// Slave ...
type Slave struct {
	cfg *config.Shared

	sess *mux.SlaveSession
}

// New ...
func New(cfg *config.Shared, conn net.Conn) (*Slave, error) {
	sess, err := mux.AcceptSession(conn)
	if err != nil {
		return nil, fmt.Errorf("mux.AcceptSession(conn): %s", err)
	}

	return &Slave{
		cfg:  cfg,
		sess: sess,
	}, nil
}

// Close ...
func (slv *Slave) Close() error {
	return slv.sess.Close()
}

// Handle ...
func (slv *Slave) Handle() error {
	for {
		m, err := slv.sess.Receive()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			log.ErrorMsg("Receiving next command: %s\n", err)
			continue
		}

		switch message := m.(type) {
		case msg.Foreground:
			slv.handleForegroundAsync(message)
		case msg.Connect:
			slv.handleConnectAsync(message)
		default:
			return fmt.Errorf("unsupported message type '%s', this is a bug", m.MsgType())
		}
	}
}
