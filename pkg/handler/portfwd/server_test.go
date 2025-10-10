package portfwd

import (
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"net"
	"testing"
)

func TestConfig_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "standard config",
			cfg: Config{
				LocalHost:  "127.0.0.1",
				LocalPort:  8080,
				RemoteHost: "192.168.1.1",
				RemotePort: 9090,
			},
			want: "PortForwarding[127.0.0.1:8080 -> 192.168.1.1:9090]",
		},
		{
			name: "localhost to localhost",
			cfg: Config{
				LocalHost:  "localhost",
				LocalPort:  3000,
				RemoteHost: "localhost",
				RemotePort: 4000,
			},
			want: "PortForwarding[localhost:3000 -> localhost:4000]",
		},
		{
			name: "wildcard address",
			cfg: Config{
				LocalHost:  "*",
				LocalPort:  1234,
				RemoteHost: "example.com",
				RemotePort: 443,
			},
			want: "PortForwarding[*:1234 -> example.com:443]",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.cfg.String()
			if got != tc.want {
				t.Errorf("Config.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := Config{
		LocalHost:  "127.0.0.1",
		LocalPort:  8080,
		RemoteHost: "192.168.1.1",
		RemotePort: 9090,
	}

	srv := NewServer(ctx, cfg, nil)

	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}
	if srv.ctx != ctx {
		t.Error("NewServer() did not set context correctly")
	}
	if srv.cfg.LocalHost != cfg.LocalHost {
		t.Errorf("NewServer() LocalHost = %q, want %q", srv.cfg.LocalHost, cfg.LocalHost)
	}
	if srv.cfg.LocalPort != cfg.LocalPort {
		t.Errorf("NewServer() LocalPort = %d, want %d", srv.cfg.LocalPort, cfg.LocalPort)
	}
}

type fakeServerControlSession struct {
	channelFn func(m msg.Message) (net.Conn, error)
}

func (f *fakeServerControlSession) SendAndGetOneChannel(m msg.Message) (net.Conn, error) {
	if f.channelFn != nil {
		return f.channelFn(m)
	}
	return nil, nil
}

func TestNewServer_AllFields(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		LocalHost:  "0.0.0.0",
		LocalPort:  12345,
		RemoteHost: "10.0.0.1",
		RemotePort: 54321,
	}

	sessCtl := &fakeServerControlSession{}
	srv := NewServer(ctx, cfg, sessCtl)

	if srv.ctx != ctx {
		t.Error("Server context not set correctly")
	}
	if srv.cfg != cfg {
		t.Error("Server config not set correctly")
	}
	if srv.sessCtl != sessCtl {
		t.Error("Server control session not set correctly")
	}
}
