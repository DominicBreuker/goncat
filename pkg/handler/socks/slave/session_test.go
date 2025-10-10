package slave

import (
	"errors"
	"testing"
)

func TestIsErrorHostUnreachable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no such host error",
			err:  errors.New("lookup example.invalid: no such host"),
			want: true,
		},
		{
			name: "connection refused",
			err:  errors.New("connect: connection refused"),
			want: false,
		},
		{
			name: "network unreachable",
			err:  errors.New("connect: network is unreachable"),
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isErrorHostUnreachable(tc.err)
			if got != tc.want {
				t.Errorf("isErrorHostUnreachable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsErrorConnectionRefused(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection refused",
			err:  errors.New("connect: connection refused"),
			want: true,
		},
		{
			name: "host is down",
			err:  errors.New("connect: host is down"),
			want: true,
		},
		{
			name: "no such host error",
			err:  errors.New("lookup example.invalid: no such host"),
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isErrorConnectionRefused(tc.err)
			if got != tc.want {
				t.Errorf("isErrorConnectionRefused() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsErrorNetworkUnreachable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "network unreachable",
			err:  errors.New("connect: network is unreachable"),
			want: true,
		},
		{
			name: "connection refused",
			err:  errors.New("connect: connection refused"),
			want: false,
		},
		{
			name: "no such host error",
			err:  errors.New("lookup example.invalid: no such host"),
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isErrorNetworkUnreachable(tc.err)
			if got != tc.want {
				t.Errorf("isErrorNetworkUnreachable() = %v, want %v", got, tc.want)
			}
		})
	}
}
