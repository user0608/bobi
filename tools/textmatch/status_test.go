package textmatch_test

import (
	"testing"

	"github.com/user0608/bobi/tools/textmatch"
)

func TestIsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "activo", input: "activo", want: true},
		{name: "activa", input: "activa", want: true},
		{name: "activos", input: "activos", want: true},
		{name: "on", input: "on", want: true},
		{name: "si", input: "si", want: true},
		{name: "si with accent", input: "sí", want: true},
		{name: "true", input: "true", want: true},
		{name: "one", input: "1", want: true},
		{name: "habilitado", input: "habilitado", want: true},
		{name: "habilitada", input: "habilitada", want: true},
		{name: "enable", input: "enable", want: true},
		{name: "enabled", input: "enabled", want: true},
		{name: "ok", input: "ok", want: true},
		{name: "operativo", input: "operativo", want: true},
		{name: "funcional", input: "funcional", want: true},
		{name: "vigente", input: "vigente", want: true},
		{name: "uppercase and surrounding spaces", input: "  ACTIVO\t", want: true},
		{name: "inactivo", input: "inactivo", want: false},
		{name: "inactiva", input: "inactiva", want: false},
		{name: "inactivos", input: "inactivos", want: false},
		{name: "off", input: "off", want: false},
		{name: "no", input: "no", want: false},
		{name: "false", input: "false", want: false},
		{name: "zero", input: "0", want: false},
		{name: "deshabilitado", input: "deshabilitado", want: false},
		{name: "deshabilitada", input: "deshabilitada", want: false},
		{name: "disable", input: "disable", want: false},
		{name: "disabled", input: "disabled", want: false},
		{name: "inoperativo", input: "inoperativo", want: false},
		{name: "no operativo", input: "no operativo", want: false},
		{name: "caducado", input: "caducado", want: false},
		{name: "suspendido", input: "suspendido", want: false},
		{name: "baja", input: "baja", want: false},
		{name: "uppercase inactive and surrounding spaces", input: "\n OFF ", want: false},
		{name: "unknown value", input: "pending", want: false},
		{name: "empty value", input: "", want: false},
		{name: "only whitespace", input: " \t\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := textmatch.IsActive(tt.input); got != tt.want {
				t.Errorf("IsActive(%q) = %t, want %t", tt.input, got, tt.want)
			}
		})
	}
}
