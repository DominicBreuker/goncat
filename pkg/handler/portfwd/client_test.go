package portfwd

import (
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"net"
	"testing"
)

type fakeClientControlSession struct {
	channelFn func() (net.Conn, error)
}

func (f *fakeClientControlSession) GetOneChannel() (net.Conn, error) {
	if f.channelFn != nil {
		return f.channelFn()
	}
	return nil, nil
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	m := msg.Connect{
		RemoteHost: "example.com",
		RemotePort: 443,
	}

	client := NewClient(ctx, m, nil)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.ctx != ctx {
		t.Error("NewClient() did not set context correctly")
	}
	if client.m.RemoteHost != m.RemoteHost {
		t.Errorf("NewClient() RemoteHost = %q, want %q", client.m.RemoteHost, m.RemoteHost)
	}
	if client.m.RemotePort != m.RemotePort {
		t.Errorf("NewClient() RemotePort = %d, want %d", client.m.RemotePort, m.RemotePort)
	}
}

func TestNewClient_AllFields(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := msg.Connect{
		RemoteHost: "192.168.1.100",
		RemotePort: 8080,
	}

	sessCtl := &fakeClientControlSession{}
	client := NewClient(ctx, m, sessCtl)

	if client.ctx != ctx {
		t.Error("Client context not set correctly")
	}
	if client.m != m {
		t.Error("Client message not set correctly")
	}
	if client.sessCtl != sessCtl {
		t.Error("Client control session not set correctly")
	}
}
