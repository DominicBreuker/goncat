package slave

import (
	"context"
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
	ctx context.Context
	cfg *config.Shared

	sess *mux.SlaveSession
}

// New ...
func New(ctx context.Context, cfg *config.Shared, conn net.Conn) (*Slave, error) {
	sess, err := mux.AcceptSession(conn)
	if err != nil {
		return nil, fmt.Errorf("mux.AcceptSession(conn): %s", err)
	}

	return &Slave{
		ctx:  ctx,
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
	ctx, cancel := context.WithCancel(slv.ctx)
	defer cancel()

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
			slv.handleForegroundAsync(ctx, message)
		case msg.Connect:
			slv.handleConnectAsync(ctx, message)
		case msg.PortFwd:
			slv.handlePortFwdAsync(ctx, message)
		case msg.SocksConnect:
			slv.handleSocksConnectAsync(ctx, message)
		case msg.SocksAssociate:
			slv.handleSocksAsociateAsync(ctx, message)
		default:
			return fmt.Errorf("unsupported message type '%s', this is a bug", m.MsgType())
		}
	}
}
