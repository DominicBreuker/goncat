package socks

import (
	"bytes"
	"testing"
)

func TestReadIPv4(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid IPv4",
			input:   []byte{192, 168, 1, 1},
			want:    "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "localhost",
			input:   []byte{127, 0, 0, 1},
			want:    "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "incomplete IPv4",
			input:   []byte{192, 168},
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
			got, err := readIPv4(r)
			if (err != nil) != tc.wantErr {
				t.Errorf("readIPv4() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if got.String() != tc.want {
					t.Errorf("readIPv4() = %q, want %q", got.String(), tc.want)
				}
			}
		})
	}
}

func TestReadIPv6(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid IPv6",
			input:   []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			want:    "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "localhost IPv6",
			input:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			want:    "::1",
			wantErr: false,
		},
		{
			name:    "incomplete IPv6",
			input:   []byte{0x20, 0x01, 0x0d, 0xb8},
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
			got, err := readIPv6(r)
			if (err != nil) != tc.wantErr {
				t.Errorf("readIPv6() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if got.String() != tc.want {
					t.Errorf("readIPv6() = %q, want %q", got.String(), tc.want)
				}
			}
		})
	}
}

func TestReadFQDN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid FQDN",
			input:   append([]byte{11}, []byte("example.com")...),
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "short FQDN",
			input:   append([]byte{3}, []byte("abc")...),
			want:    "abc",
			wantErr: false,
		},
		{
			name:    "empty FQDN",
			input:   []byte{0},
			want:    "",
			wantErr: false,
		},
		{
			name:    "incomplete FQDN - missing data",
			input:   append([]byte{11}, []byte("example")...),
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
			got, err := readFQDN(r)
			if (err != nil) != tc.wantErr {
				t.Errorf("readFQDN() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if got != tc.want {
					t.Errorf("readFQDN() = %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func TestParseAddrAndPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		wantAddr string
		wantPort uint16
		wantAtyp Atyp
		wantErr  bool
	}{
		{
			name:     "IPv4 address and port",
			input:    append([]byte{byte(AddressTypeIPv4), 192, 168, 1, 1}, []byte{0x00, 0x50}...), // port 80
			wantAddr: "192.168.1.1",
			wantPort: 80,
			wantAtyp: AddressTypeIPv4,
			wantErr:  false,
		},
		{
			name:     "IPv6 address and port",
			input:    append(append([]byte{byte(AddressTypeIPv6)}, []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}...), []byte{0x01, 0xBB}...), // port 443
			wantAddr: "2001:db8::1",
			wantPort: 443,
			wantAtyp: AddressTypeIPv6,
			wantErr:  false,
		},
		{
			name:     "FQDN and port",
			input:    append(append([]byte{byte(AddressTypeFQDN), 11}, []byte("example.com")...), []byte{0x1F, 0x90}...), // port 8080
			wantAddr: "example.com",
			wantPort: 8080,
			wantAtyp: AddressTypeFQDN,
			wantErr:  false,
		},
		{
			name:    "unsupported address type",
			input:   []byte{0xFF, 192, 168, 1, 1, 0x00, 0x50},
			wantErr: true,
		},
		{
			name:    "incomplete address",
			input:   []byte{byte(AddressTypeIPv4), 192, 168},
			wantErr: true,
		},
		{
			name:    "missing port",
			input:   []byte{byte(AddressTypeIPv4), 192, 168, 1, 1},
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
			addr, port, err := parseAddrAndPort(r)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseAddrAndPort() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			if addr.String() != tc.wantAddr {
				t.Errorf("parseAddrAndPort() addr = %q, want %q", addr.String(), tc.wantAddr)
			}
			if port != tc.wantPort {
				t.Errorf("parseAddrAndPort() port = %d, want %d", port, tc.wantPort)
			}
			if addr.Atyp() != tc.wantAtyp {
				t.Errorf("parseAddrAndPort() atyp = %v, want %v", addr.Atyp(), tc.wantAtyp)
			}
		})
	}
}
