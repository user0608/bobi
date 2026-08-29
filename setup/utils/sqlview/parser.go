package sqlview

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func parseViews(source []byte) ([]View, error) {
	tokens, err := lexSQL(source)
	if err != nil {
		return nil, err
	}

	views := []View{}
	for index := 0; index < len(tokens); index++ {
		if !isKeyword(tokens[index], "CREATE") {
			continue
		}

		view, next, matched, err := parseCreateView(tokens, index)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}

		views = append(views, view)
		index = next - 1
	}

	return views, nil
}

func parseCreateView(tokens []token, create int) (View, int, bool, error) {
	index := create + 1

	if keywordAt(tokens, index, "OR") {
		if !keywordAt(tokens, index+1, "REPLACE") {
			return View{}, create + 1, false, nil
		}
		index += 2
	}

	materialized := false
	if keywordAt(tokens, index, "MATERIALIZED") {
		materialized = true
		index++
	}
	if keywordAt(tokens, index, "TEMP") || keywordAt(tokens, index, "TEMPORARY") {
		index++
	}
	if keywordAt(tokens, index, "RECURSIVE") {
		index++
	}
	if !keywordAt(tokens, index, "VIEW") {
		return View{}, create + 1, false, nil
	}
	index++

	if keywordAt(tokens, index, "IF") {
		if !keywordAt(tokens, index+1, "NOT") || !keywordAt(tokens, index+2, "EXISTS") {
			return View{}, 0, true, parseError(tokens, index, "expected IF NOT EXISTS")
		}
		index += 3
	}

	name, index, ok, err := parseIdentifier(tokens, index)
	if err != nil {
		return View{}, 0, true, err
	}
	if !ok {
		return View{}, 0, true, parseError(tokens, index, "expected view name")
	}

	if tokenAt(tokens, index, ".") {
		index++
		viewName, next, ok, err := parseIdentifier(tokens, index)
		if err != nil {
			return View{}, 0, true, err
		}
		if !ok {
			return View{}, 0, true, parseError(tokens, index, "expected view name after schema")
		}
		name += "." + viewName
		index = next
	}

	return View{Name: name, Materialized: materialized}, index, true, nil
}

func parseIdentifier(tokens []token, index int) (string, int, bool, error) {
	if index >= len(tokens) {
		return "", index, false, nil
	}

	value := tokens[index]
	switch value.kind {
	case tokenWord, tokenQuotedIdentifier, tokenString:
		return value.text, index + 1, true, nil
	case tokenUnicodeIdentifier:
		index++
		identifier := value.text
		if !keywordAt(tokens, index, "UESCAPE") {
			return identifier, index, true, nil
		}
		if index+1 >= len(tokens) || tokens[index+1].kind != tokenString {
			return "", 0, false, parseError(tokens, index+1, "expected UESCAPE character")
		}
		if err := validateUnicodeEscape(tokens[index+1]); err != nil {
			return "", 0, false, err
		}
		return identifier + " UESCAPE " + tokens[index+1].text, index + 2, true, nil
	default:
		return "", index, false, nil
	}
}

func validateUnicodeEscape(value token) error {
	literal := value.text[1 : len(value.text)-1]
	literal = strings.ReplaceAll(literal, "''", "'")
	if literal == "" {
		return fmt.Errorf("UESCAPE must be exactly one character at byte %d", value.offset)
	}
	escape, size := utf8.DecodeRuneInString(literal)
	if escape == utf8.RuneError && size == 1 || size != len(literal) {
		return fmt.Errorf("UESCAPE must be exactly one character at byte %d", value.offset)
	}
	if unicode.IsSpace(escape) || unicode.Is(unicode.Hex_Digit, escape) || strings.ContainsRune("+'\"", escape) {
		return fmt.Errorf("invalid UESCAPE character at byte %d", value.offset)
	}
	return nil
}

func isKeyword(value token, keyword string) bool {
	return value.kind == tokenWord && strings.EqualFold(value.text, keyword)
}

func keywordAt(tokens []token, index int, keyword string) bool {
	return index < len(tokens) && isKeyword(tokens[index], keyword)
}

func tokenAt(tokens []token, index int, value string) bool {
	return index < len(tokens) && tokens[index].text == value
}

func parseError(tokens []token, index int, message string) error {
	if index >= len(tokens) {
		return fmt.Errorf("%s at end of input", message)
	}
	return fmt.Errorf("%s at byte %d", message, tokens[index].offset)
}
