package textmatch

import (
	"slices"
	"strings"
)

func IsAffirmative(input string) bool {
	v := strings.ToLower(strings.TrimSpace(input))
	affirmatives := []string{
		"yes", "y", "true", "t", "1", "on", "ok", "sure", "yeah", "affirmative",
		"si", "sí", "s", "vale", "afirmativo", "confirmo",
	}
	return slices.Contains(affirmatives, v)
}
