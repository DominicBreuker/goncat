package socks

import (
	"net/netip"
	"testing"
)

func TestCmd_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  Cmd
		want string
	}{
		{
			name: "CONNECT command",
			cmd:  CommandConnect,
			want: "CONNECT",
		},
		{
			name: "UDP ASSOCIATE command",
			cmd:  CommandAssociate,
			want: "UDP ASSOCIATE",
		},
		{
			name: "unknown command",
			cmd:  Cmd(0xFF),
			want: "unexpected",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.cmd.String()
			if got != tc.want {
				t.Errorf("Cmd.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsKnownMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method byte
		want   bool
	}{
		{
			name:   "no auth required",
			method: byte(MethodNoAuthenticationRequired),
			want:   true,
		},
		{
			name:   "no acceptable methods",
			method: byte(MethodNoAcceptableMethods),
			want:   true,
		},
		{
			name:   "unknown method",
			method: 0x01,
			want:   false,
		},
		{
			name:   "another unknown method",
			method: 0x42,
			want:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isKnownMethod(tc.method)
			if got != tc.want {
				t.Errorf("isKnownMethod(%#x) = %v, want %v", tc.method, got, tc.want)
			}
		})
	}
}

func TestAddrIPv4(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.1")
	addr := addrIPv4{IP: ip}

	if addr.String() != "192.168.1.1" {
		t.Errorf("addrIPv4.String() = %q, want %q", addr.String(), "192.168.1.1")
	}

	if addr.Atyp() != AddressTypeIPv4 {
		t.Errorf("addrIPv4.Atyp() = %v, want %v", addr.Atyp(), AddressTypeIPv4)
	}

	bytes := addr.Bytes()
	if len(bytes) != 4 {
		t.Errorf("addrIPv4.Bytes() length = %d, want 4", len(bytes))
	}

	netipAddr := addr.ToNetipAddr()
	if !netipAddr.Is4() {
		t.Error("addrIPv4.ToNetipAddr() did not return IPv4 address")
	}
	if netipAddr.String() != "192.168.1.1" {
		t.Errorf("addrIPv4.ToNetipAddr() = %q, want %q", netipAddr.String(), "192.168.1.1")
	}
}

func TestAddrIPv6(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("2001:db8::1")
	addr := addrIPv6{IP: ip}

	if addr.String() != "2001:db8::1" {
		t.Errorf("addrIPv6.String() = %q, want %q", addr.String(), "2001:db8::1")
	}

	if addr.Atyp() != AddressTypeIPv6 {
		t.Errorf("addrIPv6.Atyp() = %v, want %v", addr.Atyp(), AddressTypeIPv6)
	}

	bytes := addr.Bytes()
	if len(bytes) != 16 {
		t.Errorf("addrIPv6.Bytes() length = %d, want 16", len(bytes))
	}

	netipAddr := addr.ToNetipAddr()
	if !netipAddr.Is6() {
		t.Error("addrIPv6.ToNetipAddr() did not return IPv6 address")
	}
	if netipAddr.String() != "2001:db8::1" {
		t.Errorf("addrIPv6.ToNetipAddr() = %q, want %q", netipAddr.String(), "2001:db8::1")
	}
}

func TestAddrFQDN(t *testing.T) {
	t.Parallel()

	addr := addrFQDN{FQDN: "example.com"}

	if addr.String() != "example.com" {
		t.Errorf("addrFQDN.String() = %q, want %q", addr.String(), "example.com")
	}

	if addr.Atyp() != AddressTypeFQDN {
		t.Errorf("addrFQDN.Atyp() = %v, want %v", addr.Atyp(), AddressTypeFQDN)
	}

	bytes := addr.Bytes()
	if string(bytes) != "example.com" {
		t.Errorf("addrFQDN.Bytes() = %q, want %q", string(bytes), "example.com")
	}

	// FQDN ToNetipAddr returns zero value
	netipAddr := addr.ToNetipAddr()
	if netipAddr.IsValid() {
		t.Error("addrFQDN.ToNetipAddr() should return invalid address")
	}
}
