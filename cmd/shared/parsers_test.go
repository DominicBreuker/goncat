package shared

import (
	"dominicbreuker/goncat/pkg/config"
	"testing"
)

func TestParseTransport(t *testing.T) {
	tests := []struct {
		input    string
		protocol config.Protocol
		host     string
		port     int
		err      bool
	}{
		{input: "tcp://localhost:123", protocol: config.ProtoTCP, host: "localhost", port: 123, err: false},
		{input: "ws://localhost:123", protocol: config.ProtoWS, host: "localhost", port: 123, err: false},
		{input: "wss://localhost:123", protocol: config.ProtoWSS, host: "localhost", port: 123, err: false},
		{input: "udp://localhost:123", protocol: config.ProtoUDP, host: "localhost", port: 123, err: false},
		{input: "tcp://:123", protocol: config.ProtoTCP, host: "", port: 123, err: false},  // optional, we may want to bind all interfaces
		{input: "tcp://*:123", protocol: config.ProtoTCP, host: "", port: 123, err: false}, // also bind to all interfaces if * is provided
		{input: "udp://192.168.1.100:12345", protocol: config.ProtoUDP, host: "192.168.1.100", port: 12345, err: false},
		{input: "udp://*:12345", protocol: config.ProtoUDP, host: "", port: 12345, err: false},

		// error cases, bad protocols
		{input: "foobar://localhost:123", err: true},

		// error cases, bad ports
		{input: "tcp://localhost:-1", err: true},
		{input: "tcp://localhost:65536", err: true},
		{input: "tcp://localhost:999999999999999999", err: true},
		{input: "tcp://localhost:eighty", err: true},

		// error cases, bad format
		{input: "tcp://localhost:123:foobar", err: true},
		{input: "://localhost:123", err: true},
		{input: "localhost:123", err: true},
		{input: "tcp://localhost:", err: true},

		// error cases, stupid strings
		{input: "foobar", err: true},
		{input: "", err: true},
	}

	for _, tt := range tests {
		protocol, host, port, err := ParseTransport(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseTransport(%s) expected err=%t but was %t", tt.input, tt.err, (err != nil))
		}
		if (err != nil) || tt.err {
			continue // ignore return values
		}

		if (protocol != tt.protocol) || (host != tt.host) || (port != tt.port) {
			t.Errorf("parseTransport(%s) = %s %s %d but want %s %s %d", tt.input, protocol.String(), host, port, tt.protocol, tt.host, tt.port)
		}
	}
}
