package migrations

import "testing"

func TestParseMigrateCommand(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		action string
		ok     bool
	}{
		{
			name:   "migrate up",
			args:   []string{"migrate", "up"},
			action: "up",
			ok:     true,
		},
		{
			name:   "migrate down",
			args:   []string{"migrate", "down"},
			action: "down",
			ok:     true,
		},
		{
			name:   "migrate status",
			args:   []string{"migrate", "status"},
			action: "status",
			ok:     true,
		},
		{
			name:   "migrate script",
			args:   []string{"migrate", "script"},
			action: "script",
			ok:     true,
		},
		{
			name:   "global flag before command",
			args:   []string{"--config", "app.yml", "migrate", "up"},
			action: "up",
			ok:     true,
		},
		{
			name:   "global flag with equals",
			args:   []string{"--config=app.yml", "migrate", "down"},
			action: "down",
			ok:     true,
		},
		{
			name:   "flag after action",
			args:   []string{"migrate", "status", "--config", "app.yml"},
			action: "status",
			ok:     true,
		},
		{
			name:   "flag between command and action",
			args:   []string{"migrate", "--config", "app.yml", "up"},
			action: "up",
			ok:     true,
		},
		{
			name: "missing action",
			args: []string{"migrate"},
			ok:   false,
		},
		{
			name: "invalid action",
			args: []string{"migrate", "reset"},
			ok:   false,
		},
		{
			name: "invalid command",
			args: []string{"db", "up"},
			ok:   false,
		},
		{
			name: "only flags",
			args: []string{"--config", "app.yml"},
			ok:   false,
		},
		{
			name:   "short flag with value",
			args:   []string{"-c", "app.yml", "migrate", "up"},
			action: "up",
			ok:     true,
		},
		{
			name: "flag without value",
			args: []string{"--verbose", "migrate", "up"},
			ok:   false,
		},
		{
			name:   "double dash before command",
			args:   []string{"--", "migrate", "down"},
			action: "down",
			ok:     true,
		},
		{
			name: "double dash with flags is invalid",
			args: []string{"--", "--config", "app.yml", "migrate", "up"},
			ok:   false,
		},
		{
			name:   "flag with equals",
			args:   []string{"--config=app.yml", "migrate", "script"},
			action: "script",
			ok:     true,
		},
		{
			name:   "extra positional argument is ignored",
			args:   []string{"migrate", "up", "extra"},
			action: "up",
			ok:     true,
		},
		{
			name: "empty args",
			args: []string{},
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, ok := ParseMigrateCommand(tt.args)

			if ok != tt.ok {
				t.Fatalf("expected ok %v, got %v", tt.ok, ok)
			}

			if action != tt.action {
				t.Fatalf("expected action %q, got %q", tt.action, action)
			}
		})
	}
}
