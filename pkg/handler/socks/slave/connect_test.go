package slave

import (
	"context"
	"dominicbreuker/goncat/pkg/log"
	"dominicbreuker/goncat/pkg/mux/msg"
	"errors"
	"net"
	"testing"
)

type fakeClientControlSession struct {
	getOneChannelFn func() (net.Conn, error)
	sendFn          func(m msg.Message) error
}

func (f *fakeClientControlSession) GetOneChannel() (net.Conn, error) {
	if f.getOneChannelFn != nil {
		return f.getOneChannelFn()
	}
	return nil, errors.New("not implemented")
}

func (f *fakeClientControlSession) GetOneChannelContext(ctx context.Context) (net.Conn, error) {
	return f.GetOneChannel()
}

func (f *fakeClientControlSession) Send(m msg.Message) error {
	if f.sendFn != nil {
		return f.sendFn(m)
	}
	return nil
}

// SendContext implements the newer context-aware send. It delegates to the
// existing sendFn to preserve test behavior.
func (f *fakeClientControlSession) SendContext(ctx context.Context, m msg.Message) error {
	// If sendFn is set, call it; context is ignored for the test fake.
	if f.sendFn != nil {
		return f.sendFn(m)
	}
	return nil
}

func TestNewTCPRelay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	m := msg.SocksConnect{
		RemoteHost: "example.com",
		RemotePort: 443,
	}
	sessCtl := &fakeClientControlSession{}

	relay := NewTCPRelay(ctx, m, sessCtl, log.NewLogger(false), nil)

	if relay == nil {
		t.Fatal("NewTCPRelay() returned nil")
	}
	if relay.ctx != ctx {
		t.Error("Context not set correctly")
	}
	if relay.m.RemoteHost != m.RemoteHost {
		t.Errorf("RemoteHost = %q, want %q", relay.m.RemoteHost, m.RemoteHost)
	}
	if relay.m.RemotePort != m.RemotePort {
		t.Errorf("RemotePort = %d, want %d", relay.m.RemotePort, m.RemotePort)
	}
	if relay.sessCtl != sessCtl {
		t.Error("sessCtl not set correctly")
	}
}

func TestNewTCPRelay_DifferentHosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		port int
	}{
		{
			name: "localhost",
			host: "localhost",
			port: 8080,
		},
		{
			name: "IP address",
			host: "192.168.1.1",
			port: 9090,
		},
		{
			name: "domain",
			host: "example.org",
			port: 443,
		},
		{
			name: "high port",
			host: "test.com",
			port: 65535,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			m := msg.SocksConnect{
				RemoteHost: tc.host,
				RemotePort: tc.port,
			}
			sessCtl := &fakeClientControlSession{}

			relay := NewTCPRelay(ctx, m, sessCtl, log.NewLogger(false), nil)

			if relay == nil {
				t.Fatal("NewTCPRelay() returned nil")
			}
			if relay.m.RemoteHost != tc.host {
				t.Errorf("RemoteHost = %q, want %q", relay.m.RemoteHost, tc.host)
			}
			if relay.m.RemotePort != tc.port {
				t.Errorf("RemotePort = %d, want %d", relay.m.RemotePort, tc.port)
			}
		})
	}
}

func TestTCPRelay_Handle_GetChannelError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	m := msg.SocksConnect{
		RemoteHost: "example.com",
		RemotePort: 443,
	}

	expectedErr := errors.New("channel error")
	sessCtl := &fakeClientControlSession{
		getOneChannelFn: func() (net.Conn, error) {
			return nil, expectedErr
		},
	}

	relay := NewTCPRelay(ctx, m, sessCtl, log.NewLogger(false), nil)
	err := relay.Handle()

	if err == nil {
		t.Error("Expected error when GetOneChannel fails, got nil")
	}
}

// Ensure interface compatibility
var _ ClientControlSession = (*fakeClientControlSession)(nil)
