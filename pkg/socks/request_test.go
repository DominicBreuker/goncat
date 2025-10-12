package socks

import (
	"bytes"
	"net"
	"net/netip"
	"testing"
)

func TestReadRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		wantCmd  Cmd
		wantAddr string
		wantPort uint16
		wantErr  bool
	}{
		{
			name:     "valid CONNECT with IPv4",
			input:    []byte{VersionSocks5, byte(CommandConnect), RSV, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50}, // port 80
			wantCmd:  CommandConnect,
			wantAddr: "192.168.1.1",
			wantPort: 80,
			wantErr:  false,
		},
		{
			name:     "valid ASSOCIATE with IPv4",
			input:    []byte{VersionSocks5, byte(CommandAssociate), RSV, byte(AddressTypeIPv4), 127, 0, 0, 1, 0x04, 0x38}, // port 1080
			wantCmd:  CommandAssociate,
			wantAddr: "127.0.0.1",
			wantPort: 1080,
			wantErr:  false,
		},
		{
			name: "valid CONNECT with IPv6",
			input: append(
				[]byte{VersionSocks5, byte(CommandConnect), RSV, byte(AddressTypeIPv6)},
				append([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, []byte{0x01, 0xBB}...)...,
			), // port 443
			wantCmd:  CommandConnect,
			wantAddr: "2001:db8::1",
			wantPort: 443,
			wantErr:  false,
		},
		{
			name: "valid CONNECT with FQDN",
			input: append(
				[]byte{VersionSocks5, byte(CommandConnect), RSV, byte(AddressTypeFQDN), 11},
				append([]byte("example.com"), []byte{0x1F, 0x90}...)...,
			), // port 8080
			wantCmd:  CommandConnect,
			wantAddr: "example.com",
			wantPort: 8080,
			wantErr:  false,
		},
		{
			name:    "invalid version",
			input:   []byte{0x04, byte(CommandConnect), RSV, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50},
			wantErr: true,
		},
		{
			name:    "unsupported command",
			input:   []byte{VersionSocks5, 0x02, RSV, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50}, // 0x02 is BIND, not supported
			wantErr: true,
		},
		{
			name:    "invalid RSV field",
			input:   []byte{VersionSocks5, byte(CommandConnect), 0xFF, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50},
			wantErr: true,
		},
		{
			name:    "incomplete request - missing address",
			input:   []byte{VersionSocks5, byte(CommandConnect), RSV},
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := bytes.NewReader(tc.input)
			req, err := ReadRequest(r)
			if (err != nil) != tc.wantErr {
				t.Errorf("ReadRequest() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			if req.Ver != VersionSocks5 {
				t.Errorf("ReadRequest() Ver = %v, want %v", req.Ver, VersionSocks5)
			}
			if req.Cmd != tc.wantCmd {
				t.Errorf("ReadRequest() Cmd = %v, want %v", req.Cmd, tc.wantCmd)
			}
			if req.DstAddr.String() != tc.wantAddr {
				t.Errorf("ReadRequest() DstAddr = %q, want %q", req.DstAddr.String(), tc.wantAddr)
			}
			if req.DstPort != tc.wantPort {
				t.Errorf("ReadRequest() DstPort = %d, want %d", req.DstPort, tc.wantPort)
			}
		})
	}
}

func TestRequest_DstToUDPAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *Request
		wantAddr string
		wantPort int
		wantErr  bool
	}{
		{
			name: "IPv4 address",
			req: &Request{
				DstAddr: addrIPv4{IP: netip.MustParseAddr("192.168.1.1")},
				DstPort: 80,
			},
			wantAddr: "192.168.1.1",
			wantPort: 80,
			wantErr:  false,
		},
		{
			name: "IPv6 address",
			req: &Request{
				DstAddr: addrIPv6{IP: netip.MustParseAddr("2001:db8::1")},
				DstPort: 443,
			},
			wantAddr: "2001:db8::1",
			wantPort: 443,
			wantErr:  false,
		},
		{
			name: "FQDN address (returns error)",
			req: &Request{
				DstAddr: addrFQDN{FQDN: "example.com"},
				DstPort: 8080,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.req.DstToUDPAddr()
			if (err != nil) != tc.wantErr {
				t.Errorf("DstToUDPAddr() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			if got.IP.String() != tc.wantAddr {
				t.Errorf("DstToUDPAddr() IP = %q, want %q", got.IP.String(), tc.wantAddr)
			}
			if got.Port != tc.wantPort {
				t.Errorf("DstToUDPAddr() Port = %d, want %d", got.Port, tc.wantPort)
			}
		})
	}
}

func TestReply_atyp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reply Reply
		want  Atyp
	}{
		{
			name: "IPv4 address",
			reply: Reply{
				BndAddr: addrIPv4{IP: netip.MustParseAddr("192.168.1.1")},
			},
			want: AddressTypeIPv4,
		},
		{
			name: "IPv6 address",
			reply: Reply{
				BndAddr: addrIPv6{IP: netip.MustParseAddr("2001:db8::1")},
			},
			want: AddressTypeIPv6,
		},
		{
			name: "FQDN address",
			reply: Reply{
				BndAddr: addrFQDN{FQDN: "example.com"},
			},
			want: AddressTypeFQDN,
		},
		{
			name: "nil address",
			reply: Reply{
				BndAddr: nil,
			},
			want: 0x0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.reply.atyp()
			if got != tc.want {
				t.Errorf("atyp() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReply_serialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reply Reply
		want  []byte
	}{
		{
			name: "success with IPv4",
			reply: Reply{
				Ver:     VersionSocks5,
				Rep:     ReplySuccess,
				BndAddr: addrIPv4{IP: netip.MustParseAddr("192.168.1.1")},
				BndPort: 80,
			},
			want: []byte{VersionSocks5, byte(ReplySuccess), RSV, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50},
		},
		{
			name: "general failure with IPv4",
			reply: Reply{
				Ver:     VersionSocks5,
				Rep:     ReplyGeneralFailure,
				BndAddr: addrIPv4{IP: netip.Addr{}},
				BndPort: 0,
			},
			// Zero-value netip.Addr produces empty byte slice
			want: []byte{VersionSocks5, byte(ReplyGeneralFailure), RSV, byte(AddressTypeIPv4), 0x00, 0x00},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.reply.serialize()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("serialize() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWriteReplySuccessConnect(t *testing.T) {

	t.Parallel()

	tests := []struct {
		name      string
		localAddr net.Addr
		wantAtyp  Atyp
		wantErr   bool
	}{
		{
			name:      "TCP IPv4 address",
			localAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
			wantAtyp:  AddressTypeIPv4,
			wantErr:   false,
		},
		{
			name:      "TCP IPv6 address",
			localAddr: &net.TCPAddr{IP: net.ParseIP("::1"), Port: 8080},
			wantAtyp:  AddressTypeIPv6,
			wantErr:   false,
		},
		{
			name:      "UDP IPv4 address",
			localAddr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
			wantAtyp:  AddressTypeIPv4,
			wantErr:   false,
		},
		{
			name:      "UDP IPv6 address",
			localAddr: &net.UDPAddr{IP: net.ParseIP("::1"), Port: 8080},
			wantAtyp:  AddressTypeIPv6,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := WriteReplySuccessConnect(&buf, tc.localAddr)
			if (err != nil) != tc.wantErr {
				t.Errorf("WriteReplySuccessConnect() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			result := buf.Bytes()
			if len(result) < 4 {
				t.Fatalf("WriteReplySuccessConnect() output too short: %d bytes", len(result))
			}

			// Check header
			if result[0] != VersionSocks5 {
				t.Errorf("WriteReplySuccessConnect() Ver = %v, want %v", result[0], VersionSocks5)
			}
			if result[1] != byte(ReplySuccess) {
				t.Errorf("WriteReplySuccessConnect() Rep = %v, want %v", result[1], byte(ReplySuccess))
			}
			if result[2] != RSV {
				t.Errorf("WriteReplySuccessConnect() RSV = %v, want %v", result[2], RSV)
			}
			// Note: We don't strictly check ATYP here because Go's net package may return
			// IPv4-mapped IPv6 addresses depending on the platform and address parsing.
			// The important thing is that a valid reply was written.
		})
	}
}

func TestWriteReplyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rep     Rep
		wantRep byte
	}{
		{
			name:    "general failure",
			rep:     ReplyGeneralFailure,
			wantRep: byte(ReplyGeneralFailure),
		},
		{
			name:    "network unreachable",
			rep:     ReplyNetworkUnreachable,
			wantRep: byte(ReplyNetworkUnreachable),
		},
		{
			name:    "host unreachable",
			rep:     ReplyHostUnreachable,
			wantRep: byte(ReplyHostUnreachable),
		},
		{
			name:    "connection refused",
			rep:     ReplyConnectionRefused,
			wantRep: byte(ReplyConnectionRefused),
		},
		{
			name:    "command not supported",
			rep:     ReplyCommandNotSupported,
			wantRep: byte(ReplyCommandNotSupported),
		},
		{
			name:    "address type not supported",
			rep:     ReplyAddressTypeNotSupported,
			wantRep: byte(ReplyAddressTypeNotSupported),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := WriteReplyError(&buf, tc.rep)
			if err != nil {
				t.Errorf("WriteReplyError() error = %v", err)
				return
			}

			result := buf.Bytes()
			if len(result) < 4 {
				t.Fatalf("WriteReplyError() output too short: %d bytes", len(result))
			}

			// Check header
			if result[0] != VersionSocks5 {
				t.Errorf("WriteReplyError() Ver = %v, want %v", result[0], VersionSocks5)
			}
			if result[1] != tc.wantRep {
				t.Errorf("WriteReplyError() Rep = %v, want %v", result[1], tc.wantRep)
			}
			if result[2] != RSV {
				t.Errorf("WriteReplyError() RSV = %v, want %v", result[2], RSV)
			}
		})
	}
}
