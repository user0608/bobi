package textmatch_test

import (
	"testing"

	"github.com/user0608/bobi/tools/textmatch"
)

func TestIsAffirmative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "yes", input: "yes", want: true},
		{name: "y", input: "y", want: true},
		{name: "true", input: "true", want: true},
		{name: "t", input: "t", want: true},
		{name: "one", input: "1", want: true},
		{name: "on", input: "on", want: true},
		{name: "ok", input: "ok", want: true},
		{name: "sure", input: "sure", want: true},
		{name: "yeah", input: "yeah", want: true},
		{name: "affirmative", input: "affirmative", want: true},
		{name: "si", input: "si", want: true},
		{name: "si with accent", input: "sí", want: true},
		{name: "s", input: "s", want: true},
		{name: "vale", input: "vale", want: true},
		{name: "afirmativo", input: "afirmativo", want: true},
		{name: "confirmo", input: "confirmo", want: true},
		{name: "uppercase and surrounding spaces", input: "  SÍ\t", want: true},
		{name: "negative value", input: "no", want: false},
		{name: "unknown value", input: "maybe", want: false},
		{name: "partial match", input: "yes please", want: false},
		{name: "empty value", input: "", want: false},
		{name: "only whitespace", input: " \t\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := textmatch.IsAffirmative(tt.input); got != tt.want {
				t.Errorf("IsAffirmative(%q) = %t, want %t", tt.input, got, tt.want)
			}
		})
	}
}
