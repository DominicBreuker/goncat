package format

import (
	"testing"
)

func TestAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{
			name: "IPv4 address",
			host: "192.168.1.1",
			port: 8080,
			want: "192.168.1.1:8080",
		},
		{
			name: "IPv4 localhost",
			host: "127.0.0.1",
			port: 80,
			want: "127.0.0.1:80",
		},
		{
			name: "hostname",
			host: "example.com",
			port: 443,
			want: "example.com:443",
		},
		{
			name: "IPv6 address",
			host: "::1",
			port: 8080,
			want: "[::1]:8080",
		},
		{
			name: "IPv6 full address",
			host: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			port: 443,
			want: "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:443",
		},
		{
			name: "IPv6 compressed",
			host: "2001:db8::1",
			port: 80,
			want: "[2001:db8::1]:80",
		},
		{
			name: "wildcard",
			host: "*",
			port: 8080,
			want: "*:8080",
		},
		{
			name: "empty host",
			host: "",
			port: 8080,
			want: ":8080",
		},
		{
			name: "port 1",
			host: "localhost",
			port: 1,
			want: "localhost:1",
		},
		{
			name: "port 65535",
			host: "localhost",
			port: 65535,
			want: "localhost:65535",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Addr(tc.host, tc.port)
			if got != tc.want {
				t.Errorf("Addr(%q, %d) = %q, want %q", tc.host, tc.port, got, tc.want)
			}
		})
	}
}
