package socks

import (
	"bytes"
	"net"
	"net/netip"
	"testing"
)

func TestUDPRequest_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  *UDPRequest
		want string
	}{
		{
			name: "basic UDP request",
			req: &UDPRequest{
				Frag:    0,
				DstAddr: addrIPv4{IP: netip.MustParseAddr("192.168.1.1")},
				DstPort: 80,
				Data:    []byte("test data"),
			},
			want: "Datagram[0|192.168.1.1:80|test data]",
		},
		{
			name: "empty data",
			req: &UDPRequest{
				Frag:    0,
				DstAddr: addrIPv4{IP: netip.MustParseAddr("127.0.0.1")},
				DstPort: 1080,
				Data:    []byte{},
			},
			want: "Datagram[0|127.0.0.1:1080|]",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.req.String()
			if got != tc.want {
				t.Errorf("UDPRequest.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadUDPDatagram(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		wantAddr string
		wantPort uint16
		wantData []byte
		wantErr  bool
	}{
		{
			name: "valid IPv4 datagram",
			input: append(
				[]byte{0x00, 0x00, 0x00, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50},
				[]byte("test data")...,
			), // RSV, RSV, FRAG, ATYP, IPv4, Port 80, Data
			wantAddr: "192.168.1.1",
			wantPort: 80,
			wantData: []byte("test data"),
			wantErr:  false,
		},
		{
			name: "valid IPv6 datagram",
			input: append(
				append(
					[]byte{0x00, 0x00, 0x00, byte(AddressTypeIPv6)},
					[]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}...,
				),
				append([]byte{0x01, 0xBB}, []byte("test")...)...,
			), // Port 443
			wantAddr: "2001:db8::1",
			wantPort: 443,
			wantData: []byte("test"),
			wantErr:  false,
		},
		{
			name: "valid FQDN datagram",
			input: append(
				append(
					[]byte{0x00, 0x00, 0x00, byte(AddressTypeFQDN), 11},
					append([]byte("example.com"), []byte{0x1F, 0x90}...)...,
				),
				[]byte("data")...,
			), // Port 8080
			wantAddr: "example.com",
			wantPort: 8080,
			wantData: []byte("data"),
			wantErr:  false,
		},
		{
			name:    "invalid RSV field",
			input:   []byte{0x01, 0x00, 0x00, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50},
			wantErr: true,
		},
		{
			name:    "invalid FRAG value (> 127)",
			input:   []byte{0x00, 0x00, 0x80, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50},
			wantErr: true,
		},
		{
			name:    "fragmentation not supported",
			input:   []byte{0x00, 0x00, 0x01, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50},
			wantErr: true,
		},
		{
			name:    "unsupported address type",
			input:   []byte{0x00, 0x00, 0x00, 0xFF, 192, 168, 1, 1, 0x00, 0x50},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ReadUDPDatagram(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ReadUDPDatagram() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			if got.DstAddr.String() != tc.wantAddr {
				t.Errorf("ReadUDPDatagram() DstAddr = %q, want %q", got.DstAddr.String(), tc.wantAddr)
			}
			if got.DstPort != tc.wantPort {
				t.Errorf("ReadUDPDatagram() DstPort = %d, want %d", got.DstPort, tc.wantPort)
			}
			if !bytes.Equal(got.Data, tc.wantData) {
				t.Errorf("ReadUDPDatagram() Data = %v, want %v", got.Data, tc.wantData)
			}
		})
	}
}

func TestUDPRequest_serialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  UDPRequest
		want []byte
	}{
		{
			name: "IPv4 datagram",
			req: UDPRequest{
				Frag:    0,
				DstAddr: addrIPv4{IP: netip.MustParseAddr("192.168.1.1")},
				DstPort: 80,
				Data:    []byte("test"),
			},
			want: []byte{0x00, 0x00, 0x00, byte(AddressTypeIPv4), 192, 168, 1, 1, 0x00, 0x50, 't', 'e', 's', 't'},
		},
		{
			name: "IPv6 datagram",
			req: UDPRequest{
				Frag:    0,
				DstAddr: addrIPv6{IP: netip.MustParseAddr("2001:db8::1")},
				DstPort: 443,
				Data:    []byte("data"),
			},
			want: append(
				append(
					[]byte{0x00, 0x00, 0x00, byte(AddressTypeIPv6)},
					[]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}...,
				),
				append([]byte{0x01, 0xBB}, []byte("data")...)...,
			),
		},
		{
			name: "empty data",
			req: UDPRequest{
				Frag:    0,
				DstAddr: addrIPv4{IP: netip.MustParseAddr("127.0.0.1")},
				DstPort: 1080,
				Data:    []byte{},
			},
			want: []byte{0x00, 0x00, 0x00, byte(AddressTypeIPv4), 127, 0, 0, 1, 0x04, 0x38},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.req.serialize()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("serialize() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWriteUDPRequestAddrPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	t.Parallel()

	// Create a local UDP server to accept the datagrams
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve server address: %v", err)
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to create server connection: %v", err)
	}
	defer serverConn.Close()

	serverIP := netip.MustParseAddr("127.0.0.1")
	serverPort := uint16(serverConn.LocalAddr().(*net.UDPAddr).Port)

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "with data",
			data:    []byte("test"),
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create an unconnected client connection (required for WriteToUDPAddrPort)
			clientAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("Failed to resolve client address: %v", err)
			}

			clientConn, err := net.ListenUDP("udp", clientAddr)
			if err != nil {
				t.Fatalf("Failed to create client connection: %v", err)
			}
			defer clientConn.Close()

			err = WriteUDPRequestAddrPort(clientConn, serverIP, serverPort, tc.data)
			if (err != nil) != tc.wantErr {
				t.Errorf("WriteUDPRequestAddrPort() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
