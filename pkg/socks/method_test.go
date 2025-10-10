package socks

import (
	"bytes"
	"testing"
)

func TestMethodSelectionRequest_IsNoAuthRequested(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		methods []Method
		want    bool
	}{
		{
			name:    "no auth requested",
			methods: []Method{MethodNoAuthenticationRequired},
			want:    true,
		},
		{
			name:    "no auth among multiple methods",
			methods: []Method{Method(0x01), MethodNoAuthenticationRequired, Method(0x02)},
			want:    true,
		},
		{
			name:    "no auth not requested",
			methods: []Method{Method(0x01), Method(0x02)},
			want:    false,
		},
		{
			name:    "empty methods",
			methods: []Method{},
			want:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msr := &MethodSelectionRequest{
				Ver:      VersionSocks5,
				NMethods: byte(len(tc.methods)),
				Methods:  tc.methods,
			}
			got := msr.IsNoAuthRequested()
			if got != tc.want {
				t.Errorf("IsNoAuthRequested() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReadMethodSelectionRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       []byte
		wantVer     byte
		wantMethods []Method
		wantErr     bool
	}{
		{
			name:        "valid request with no auth",
			input:       []byte{VersionSocks5, 1, byte(MethodNoAuthenticationRequired)},
			wantVer:     VersionSocks5,
			wantMethods: []Method{MethodNoAuthenticationRequired},
			wantErr:     false,
		},
		{
			name:        "valid request with multiple methods",
			input:       []byte{VersionSocks5, 2, byte(MethodNoAuthenticationRequired), byte(MethodNoAcceptableMethods)},
			wantVer:     VersionSocks5,
			wantMethods: []Method{MethodNoAuthenticationRequired, MethodNoAcceptableMethods},
			wantErr:     false,
		},
		{
			name:    "invalid version",
			input:   []byte{0x04, 1, byte(MethodNoAuthenticationRequired)},
			wantErr: true,
		},
		{
			name:    "incomplete request - missing methods",
			input:   []byte{VersionSocks5, 2, byte(MethodNoAuthenticationRequired)},
			wantErr: true,
		},
		{
			name:    "incomplete request - missing nmethods",
			input:   []byte{VersionSocks5},
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:        "unknown methods are filtered out",
			input:       []byte{VersionSocks5, 3, byte(MethodNoAuthenticationRequired), 0x42, byte(MethodNoAcceptableMethods)},
			wantVer:     VersionSocks5,
			wantMethods: []Method{MethodNoAuthenticationRequired, MethodNoAcceptableMethods},
			wantErr:     false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := bytes.NewReader(tc.input)
			got, err := ReadMethodSelectionRequest(r)
			if (err != nil) != tc.wantErr {
				t.Errorf("ReadMethodSelectionRequest() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			if got.Ver != tc.wantVer {
				t.Errorf("ReadMethodSelectionRequest() Ver = %v, want %v", got.Ver, tc.wantVer)
			}
			if len(got.Methods) != len(tc.wantMethods) {
				t.Errorf("ReadMethodSelectionRequest() Methods length = %d, want %d", len(got.Methods), len(tc.wantMethods))
			}
			for i, m := range got.Methods {
				if i >= len(tc.wantMethods) {
					break
				}
				if m != tc.wantMethods[i] {
					t.Errorf("ReadMethodSelectionRequest() Methods[%d] = %v, want %v", i, m, tc.wantMethods[i])
				}
			}
		})
	}
}

func TestWriteMethodSelectionResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     Method
		wantOutput []byte
		wantErr    bool
	}{
		{
			name:       "no auth method",
			method:     MethodNoAuthenticationRequired,
			wantOutput: []byte{VersionSocks5, byte(MethodNoAuthenticationRequired)},
			wantErr:    false,
		},
		{
			name:       "no acceptable methods",
			method:     MethodNoAcceptableMethods,
			wantOutput: []byte{VersionSocks5, byte(MethodNoAcceptableMethods)},
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := WriteMethodSelectionResponse(&buf, tc.method)
			if (err != nil) != tc.wantErr {
				t.Errorf("WriteMethodSelectionResponse() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.wantOutput) {
				t.Errorf("WriteMethodSelectionResponse() = %v, want %v", got, tc.wantOutput)
			}
		})
	}
}

func TestMethodSelectionResponse_serialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		resp MethodSelectionResponse
		want []byte
	}{
		{
			name: "no auth method",
			resp: MethodSelectionResponse{
				Ver:    VersionSocks5,
				Method: MethodNoAuthenticationRequired,
			},
			want: []byte{VersionSocks5, byte(MethodNoAuthenticationRequired)},
		},
		{
			name: "no acceptable methods",
			resp: MethodSelectionResponse{
				Ver:    VersionSocks5,
				Method: MethodNoAcceptableMethods,
			},
			want: []byte{VersionSocks5, byte(MethodNoAcceptableMethods)},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.resp.serialize()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("serialize() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReadMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		n       int
		want    []Method
		wantErr bool
	}{
		{
			name:    "single known method",
			input:   []byte{byte(MethodNoAuthenticationRequired)},
			n:       1,
			want:    []Method{MethodNoAuthenticationRequired},
			wantErr: false,
		},
		{
			name:    "multiple known methods",
			input:   []byte{byte(MethodNoAuthenticationRequired), byte(MethodNoAcceptableMethods)},
			n:       2,
			want:    []Method{MethodNoAuthenticationRequired, MethodNoAcceptableMethods},
			wantErr: false,
		},
		{
			name:    "mixed known and unknown methods",
			input:   []byte{byte(MethodNoAuthenticationRequired), 0x42, byte(MethodNoAcceptableMethods)},
			n:       3,
			want:    []Method{MethodNoAuthenticationRequired, MethodNoAcceptableMethods},
			wantErr: false,
		},
		{
			name:    "insufficient data",
			input:   []byte{byte(MethodNoAuthenticationRequired)},
			n:       2,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			n:       1,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := bytes.NewReader(tc.input)
			got, err := readMethods(r, tc.n)
			if (err != nil) != tc.wantErr {
				t.Errorf("readMethods() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}

			if len(got) != len(tc.want) {
				t.Errorf("readMethods() length = %d, want %d", len(got), len(tc.want))
			}
			for i, m := range got {
				if i >= len(tc.want) {
					break
				}
				if m != tc.want[i] {
					t.Errorf("readMethods()[%d] = %v, want %v", i, m, tc.want[i])
				}
			}
		})
	}
}
