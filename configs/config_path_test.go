package configs_test

import (
	"testing"

	"github.com/user0608/bobi/configs"
)

func TestConfigPathFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want configs.ConfigPath
	}{
		{
			name: "long flag",
			args: []string{"--config", "config.yaml"},
			want: "config.yaml",
		},
		{
			name: "short flag",
			args: []string{"-c", "config.yaml"},
			want: "config.yaml",
		},
		{
			name: "long flag with equals",
			args: []string{"--config=config.yaml"},
			want: "config.yaml",
		},
		{
			name: "short flag with equals",
			args: []string{"-c=config.yaml"},
			want: "config.yaml",
		},
		{
			name: "trims arguments and path",
			args: []string{"  --config  ", "  config.yaml  "},
			want: "config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := configs.ConfigPathFromArgs(tt.args)
			if err != nil {
				t.Fatalf("ConfigPathFromArgs() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ConfigPathFromArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigPathFromArgsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "flag not provided", args: []string{"--verbose"}, want: "config flag not provided"},
		{name: "missing long flag value", args: []string{"--config"}, want: "missing value for config flag"},
		{name: "missing short flag value", args: []string{"-c"}, want: "missing value for config flag"},
		{name: "empty long flag value", args: []string{"--config", "   "}, want: "empty config path"},
		{name: "empty short flag value", args: []string{"-c", "   "}, want: "empty config path"},
		{name: "empty long equals value", args: []string{"--config="}, want: "empty config path"},
		{name: "empty short equals value", args: []string{"-c=   "}, want: "empty config path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := configs.ConfigPathFromArgs(tt.args)
			if err == nil {
				t.Fatal("ConfigPathFromArgs() error = nil, want error")
			}
			if err.Error() != tt.want {
				t.Fatalf("ConfigPathFromArgs() error = %q, want %q", err, tt.want)
			}
		})
	}
}
