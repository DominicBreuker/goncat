package msg

import (
	"testing"
)

// TestMessageInterface verifies that all message types implement the Message interface.
func TestMessageInterface(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  Message
		want string
	}{
		{"Connect", Connect{RemoteHost: "host", RemotePort: 80}, "Connect"},
		{"Foreground", Foreground{Exec: "/bin/sh", Pty: true}, "Foreground"},
		{"PortFwd", PortFwd{LocalHost: "local", LocalPort: 8080, RemoteHost: "remote", RemotePort: 9090}, "PortFwd"},
		{"SocksAssociate", SocksAssociate{}, "SocksAssociate"},
		{"SocksConnect", SocksConnect{RemoteHost: "host", RemotePort: 1080}, "SocksConnect"},
		{"SocksDatagram", SocksDatagram{Addr: "addr", Port: 53, Data: []byte("data")}, "SocksDatagram"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.msg.MsgType()
			if got != tc.want {
				t.Errorf("MsgType() = %q; want %q", got, tc.want)
			}
		})
	}
}

// TestConnect_MsgType verifies Connect message type string.
func TestConnect_MsgType(t *testing.T) {
	t.Parallel()
	msg := Connect{RemoteHost: "example.com", RemotePort: 80}
	if got := msg.MsgType(); got != "Connect" {
		t.Errorf("MsgType() = %q; want %q", got, "Connect")
	}
}

// TestForeground_MsgType verifies Foreground message type string.
func TestForeground_MsgType(t *testing.T) {
	t.Parallel()
	msg := Foreground{Exec: "/bin/bash", Pty: true}
	if got := msg.MsgType(); got != "Foreground" {
		t.Errorf("MsgType() = %q; want %q", got, "Foreground")
	}
}

// TestPortFwd_MsgType verifies PortFwd message type string.
func TestPortFwd_MsgType(t *testing.T) {
	t.Parallel()
	msg := PortFwd{LocalHost: "localhost", LocalPort: 8080, RemoteHost: "remote", RemotePort: 9090}
	if got := msg.MsgType(); got != "PortFwd" {
		t.Errorf("MsgType() = %q; want %q", got, "PortFwd")
	}
}

// TestSocksAssociate_MsgType verifies SocksAssociate message type string.
func TestSocksAssociate_MsgType(t *testing.T) {
	t.Parallel()
	msg := SocksAssociate{}
	if got := msg.MsgType(); got != "SocksAssociate" {
		t.Errorf("MsgType() = %q; want %q", got, "SocksAssociate")
	}
}

// TestSocksConnect_MsgType verifies SocksConnect message type string.
func TestSocksConnect_MsgType(t *testing.T) {
	t.Parallel()
	msg := SocksConnect{RemoteHost: "proxy.example.com", RemotePort: 1080}
	if got := msg.MsgType(); got != "SocksConnect" {
		t.Errorf("MsgType() = %q; want %q", got, "SocksConnect")
	}
}

// TestSocksDatagram_MsgType verifies SocksDatagram message type string.
func TestSocksDatagram_MsgType(t *testing.T) {
	t.Parallel()
	msg := SocksDatagram{Addr: "8.8.8.8", Port: 53, Data: []byte("test")}
	if got := msg.MsgType(); got != "SocksDatagram" {
		t.Errorf("MsgType() = %q; want %q", got, "SocksDatagram")
	}
}

// TestSocksDatagram_String verifies SocksDatagram string representation.
func TestSocksDatagram_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  SocksDatagram
		want string
	}{
		{
			name: "basic datagram",
			msg:  SocksDatagram{Addr: "8.8.8.8", Port: 53, Data: []byte("test")},
			want: "Datagram[4|8.8.8.8|53|test]",
		},
		{
			name: "empty data",
			msg:  SocksDatagram{Addr: "1.1.1.1", Port: 80, Data: []byte("")},
			want: "Datagram[0|1.1.1.1|80|]",
		},
		{
			name: "binary data",
			msg:  SocksDatagram{Addr: "host", Port: 443, Data: []byte{0x01, 0x02, 0x03}},
			want: "Datagram[3|host|443|\x01\x02\x03]",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.msg.String()
			if got != tc.want {
				t.Errorf("String() = %q; want %q", got, tc.want)
			}
		})
	}
}
