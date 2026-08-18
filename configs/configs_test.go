package configs_test

import (
	"os"
	"strings"
	"testing"

	"github.com/user0608/bobi/configs"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	configFile := t.TempDir() + "/config.yaml"
	if err := writeFile(configFile, "name: bobi\nport: 8080\n"); err != nil {
		t.Fatal(err)
	}

	got, err := configs.LoadConfig(configs.ConfigPath("  " + configFile + "  "))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got.GetString("name") != "bobi" {
		t.Fatalf("LoadConfig() name = %q, want %q", got.GetString("name"), "bobi")
	}
	if got.GetInt("port") != 8080 {
		t.Fatalf("LoadConfig() port = %d, want %d", got.GetInt("port"), 8080)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path configs.ConfigPath
		want string
	}{
		{name: "empty path", path: "   ", want: "config path is empty"},
		{name: "missing file", path: "/path/that/does/not/exist/config.yaml", want: "read config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := configs.LoadConfig(tt.path)
			if got != nil {
				t.Fatal("LoadConfig() config != nil, want nil")
			}
			if err == nil {
				t.Fatal("LoadConfig() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadConfig() error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
