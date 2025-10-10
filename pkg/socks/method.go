package socks

import (
	"fmt"
	"io"
)

// ######## Request ######## //
//
// https://datatracker.ietf.org/doc/html/rfc1928#section-3
//
//                   +----+----------+----------+
//                   |VER | NMETHODS | METHODS  |
//                   +----+----------+----------+
//                   | 1  |    1     | 1 to 255 |
//                   +----+----------+----------+

// MethodSelectionRequest is sent by a SOCKS client to the server to initiate the connection.
// Clients specify a version and the methods they support.
type MethodSelectionRequest struct {
	Ver      byte
	NMethods byte
	Methods  []Method
}

// IsNoAuthRequested returns true if the client supports connections without authentication.
func (msr *MethodSelectionRequest) IsNoAuthRequested() bool {
	for _, m := range msr.Methods {
		if m == MethodNoAuthenticationRequired {
			return true
		}
	}

	return false
}

// ReadMethodSelectionRequest reads a complete method selection request from r.
func ReadMethodSelectionRequest(r io.Reader) (*MethodSelectionRequest, error) {
	var out MethodSelectionRequest
	var err error

	b := []byte{0}
	if _, err = r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing version: %s", err)
	}
	out.Ver = b[0]

	if out.Ver != VersionSocks5 {
		return nil, fmt.Errorf("requested version was %d but only SOCKS5 (%d) supported", out.Ver, VersionSocks5)
	}

	if _, err = r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing number of methods: %s", err)
	}
	out.NMethods = b[0]

	out.Methods, err = readMethods(r, int(out.NMethods))
	if err != nil {
		return nil, fmt.Errorf("reading methods: %s", err)
	}

	return &out, nil
}

func readMethods(r io.Reader, n int) ([]Method, error) {
	bs := make([]byte, n)
	_, err := io.ReadAtLeast(r, bs, n)
	if err != nil {
		return nil, fmt.Errorf("parsing %d methods: %s", n, err)
	}

	var out []Method
	for _, b := range bs {
		if isKnownMethod(b) {
			out = append(out, Method(b))
		}
	}
	return out, nil
}

// ######## Response ######## //
//
// https://datatracker.ietf.org/doc/html/rfc1928#section-3
//
//                         +----+--------+
//                         |VER | METHOD |
//                         +----+--------+
//                         | 1  |   1    |
//                         +----+--------+

// MethodSelectionResponse is the response a server sends to a method selection request.
// The server indicates to the client which method it selected.
type MethodSelectionResponse struct {
	Ver    byte
	Method Method
}

func (msr MethodSelectionResponse) serialize() []byte {
	return []byte{msr.Ver, byte(msr.Method)}
}

// WriteMethodSelectionResponse writes a complete serialized method selection response to w.
func WriteMethodSelectionResponse(w io.Writer, method Method) error {
	resp := MethodSelectionResponse{
		Ver:    VersionSocks5,
		Method: method,
	}

	_, err := w.Write(resp.serialize())
	if err != nil {
		return fmt.Errorf("writing serialized response: %s", err)
	}

	return nil
}
