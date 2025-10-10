package msg

import (
	"bytes"
	"encoding/gob"
	"testing"
)

// TestGobEncoding verifies that all message types can be encoded and decoded via gob.
func TestGobEncoding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  Message
	}{
		{"Connect", Connect{RemoteHost: "example.com", RemotePort: 8080}},
		{"Foreground", Foreground{Exec: "/bin/bash", Pty: true}},
		{"Foreground_no_pty", Foreground{Exec: "/bin/sh", Pty: false}},
		{"Foreground_empty_exec", Foreground{Exec: "", Pty: false}},
		{"PortFwd", PortFwd{LocalHost: "localhost", LocalPort: 3000, RemoteHost: "remote", RemotePort: 4000}},
		{"SocksAssociate", SocksAssociate{}},
		{"SocksConnect", SocksConnect{RemoteHost: "proxy.com", RemotePort: 1080}},
		{"SocksDatagram", SocksDatagram{Addr: "8.8.8.8", Port: 53, Data: []byte("test data")}},
		{"SocksDatagram_empty", SocksDatagram{Addr: "host", Port: 80, Data: []byte{}}},
		{"SocksDatagram_binary", SocksDatagram{Addr: "host", Port: 443, Data: []byte{0x00, 0x01, 0x02, 0xff}}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Encode
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(&tc.msg); err != nil {
				t.Fatalf("Encode() failed: %v", err)
			}

			// Decode
			dec := gob.NewDecoder(&buf)
			var decoded Message
			if err := dec.Decode(&decoded); err != nil {
				t.Fatalf("Decode() failed: %v", err)
			}

			// Verify type matches
			if decoded.MsgType() != tc.msg.MsgType() {
				t.Errorf("MsgType mismatch: got %q, want %q", decoded.MsgType(), tc.msg.MsgType())
			}

			// Type-specific comparisons
			switch original := tc.msg.(type) {
			case Connect:
				result, ok := decoded.(Connect)
				if !ok {
					t.Fatalf("decoded type = %T; want Connect", decoded)
				}
				if result.RemoteHost != original.RemoteHost || result.RemotePort != original.RemotePort {
					t.Errorf("Connect mismatch: got %+v, want %+v", result, original)
				}

			case Foreground:
				result, ok := decoded.(Foreground)
				if !ok {
					t.Fatalf("decoded type = %T; want Foreground", decoded)
				}
				if result.Exec != original.Exec || result.Pty != original.Pty {
					t.Errorf("Foreground mismatch: got %+v, want %+v", result, original)
				}

			case PortFwd:
				result, ok := decoded.(PortFwd)
				if !ok {
					t.Fatalf("decoded type = %T; want PortFwd", decoded)
				}
				if result.LocalHost != original.LocalHost || result.LocalPort != original.LocalPort ||
					result.RemoteHost != original.RemoteHost || result.RemotePort != original.RemotePort {
					t.Errorf("PortFwd mismatch: got %+v, want %+v", result, original)
				}

			case SocksAssociate:
				_, ok := decoded.(SocksAssociate)
				if !ok {
					t.Fatalf("decoded type = %T; want SocksAssociate", decoded)
				}

			case SocksConnect:
				result, ok := decoded.(SocksConnect)
				if !ok {
					t.Fatalf("decoded type = %T; want SocksConnect", decoded)
				}
				if result.RemoteHost != original.RemoteHost || result.RemotePort != original.RemotePort {
					t.Errorf("SocksConnect mismatch: got %+v, want %+v", result, original)
				}

			case SocksDatagram:
				result, ok := decoded.(SocksDatagram)
				if !ok {
					t.Fatalf("decoded type = %T; want SocksDatagram", decoded)
				}
				if result.Addr != original.Addr || result.Port != original.Port {
					t.Errorf("SocksDatagram address/port mismatch: got %+v, want %+v", result, original)
				}
				if !bytes.Equal(result.Data, original.Data) {
					t.Errorf("SocksDatagram data mismatch: got %v, want %v", result.Data, original.Data)
				}
			}
		})
	}
}

// TestGobEncoding_MultipleMessages verifies multiple messages can be encoded/decoded in sequence.
func TestGobEncoding_MultipleMessages(t *testing.T) {
	t.Parallel()

	messages := []Message{
		Connect{RemoteHost: "host1", RemotePort: 80},
		Foreground{Exec: "/bin/sh", Pty: true},
		SocksConnect{RemoteHost: "host2", RemotePort: 443},
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)

	// Encode all messages
	for i, msg := range messages {
		if err := enc.Encode(&msg); err != nil {
			t.Fatalf("Encode() message %d failed: %v", i, err)
		}
	}

	// Decode all messages
	for i, want := range messages {
		var got Message
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("Decode() message %d failed: %v", i, err)
		}
		if got.MsgType() != want.MsgType() {
			t.Errorf("message %d: MsgType = %q; want %q", i, got.MsgType(), want.MsgType())
		}
	}
}
