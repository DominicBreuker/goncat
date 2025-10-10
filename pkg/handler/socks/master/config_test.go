package master

import (
	"testing"
)

func TestConfig_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "localhost with standard port",
			cfg: Config{
				LocalHost: "127.0.0.1",
				LocalPort: 1080,
			},
			want: "127.0.0.1:1080",
		},
		{
			name: "wildcard address",
			cfg: Config{
				LocalHost: "0.0.0.0",
				LocalPort: 8080,
			},
			want: "0.0.0.0:8080",
		},
		{
			name: "hostname",
			cfg: Config{
				LocalHost: "localhost",
				LocalPort: 9050,
			},
			want: "localhost:9050",
		},
		{
			name: "high port number",
			cfg: Config{
				LocalHost: "192.168.1.1",
				LocalPort: 54321,
			},
			want: "192.168.1.1:54321",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.cfg.String()
			if got != tc.want {
				t.Errorf("Config.String() = %q, want %q", got, tc.want)
			}
		})
	}
}
