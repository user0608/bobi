package sqlview

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

type tokenKind uint8

const (
	tokenOther tokenKind = iota
	tokenWord
	tokenQuotedIdentifier
	tokenUnicodeIdentifier
	tokenString
)

type token struct {
	kind   tokenKind
	text   string
	offset int
}

func lexSQL(source []byte) ([]token, error) {
	tokens := []token{}

	for offset := 0; offset < len(source); {
		switch {
		case bytes.HasPrefix(source[offset:], []byte{0xef, 0xbb, 0xbf}):
			offset += 3
		case isSpace(source[offset]):
			offset++
		case bytes.HasPrefix(source[offset:], []byte("--")):
			offset = skipLineComment(source, offset+2)
		case bytes.HasPrefix(source[offset:], []byte("/*")):
			next, err := skipBlockComment(source, offset)
			if err != nil {
				return nil, err
			}
			offset = next
		case isPrefixedString(source, offset):
			quote := offset + 1
			if source[offset+1] == '&' {
				quote++
			}
			next, err := skipSingleQuoted(source, quote, source[offset] == 'e' || source[offset] == 'E')
			if err != nil {
				return nil, err
			}
			offset = next
		case source[offset] == '\'':
			next, err := skipSingleQuoted(source, offset, false)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{
				kind:   tokenString,
				text:   string(source[offset:next]),
				offset: offset,
			})
			offset = next
		case source[offset] == '$':
			next, matched, err := skipDollarQuoted(source, offset)
			if err != nil {
				return nil, err
			}
			if matched {
				offset = next
				continue
			}
			tokens = append(tokens, token{kind: tokenOther, text: "$", offset: offset})
			offset++
		case isUnicodeQuotedIdentifier(source, offset):
			next, err := scanDelimited(source, offset+2, '"', '"')
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{
				kind:   tokenUnicodeIdentifier,
				text:   string(source[offset:next]),
				offset: offset,
			})
			offset = next
		case source[offset] == '"' || source[offset] == '`':
			next, err := scanDelimited(source, offset, source[offset], source[offset])
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{
				kind:   tokenQuotedIdentifier,
				text:   string(source[offset:next]),
				offset: offset,
			})
			offset = next
		case source[offset] == '[':
			next, err := scanDelimited(source, offset, '[', ']')
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{
				kind:   tokenQuotedIdentifier,
				text:   string(source[offset:next]),
				offset: offset,
			})
			offset = next
		default:
			r, size := utf8.DecodeRune(source[offset:])
			if r == utf8.RuneError && size == 1 {
				return nil, fmt.Errorf("invalid UTF-8 at byte %d", offset)
			}
			if isIdentifierStart(r) {
				next := scanWord(source, offset+size)
				tokens = append(tokens, token{
					kind:   tokenWord,
					text:   string(source[offset:next]),
					offset: offset,
				})
				offset = next
				continue
			}

			tokens = append(tokens, token{
				kind:   tokenOther,
				text:   string(source[offset : offset+size]),
				offset: offset,
			})
			offset += size
		}
	}

	return tokens, nil
}

func skipLineComment(source []byte, offset int) int {
	for offset < len(source) && source[offset] != '\n' {
		offset++
	}
	return offset
}

func skipBlockComment(source []byte, offset int) (int, error) {
	start := offset
	depth := 1
	offset += 2

	for offset < len(source) {
		switch {
		case bytes.HasPrefix(source[offset:], []byte("/*")):
			depth++
			offset += 2
		case bytes.HasPrefix(source[offset:], []byte("*/")):
			depth--
			offset += 2
			if depth == 0 {
				return offset, nil
			}
		default:
			offset++
		}
	}

	return 0, fmt.Errorf("unterminated block comment at byte %d", start)
}

func skipSingleQuoted(source []byte, quote int, backslashEscapes bool) (int, error) {
	for offset := quote + 1; offset < len(source); offset++ {
		if backslashEscapes && source[offset] == '\\' {
			offset++
			continue
		}
		if source[offset] != '\'' {
			continue
		}
		if offset+1 < len(source) && source[offset+1] == '\'' {
			offset++
			continue
		}
		return offset + 1, nil
	}

	return 0, fmt.Errorf("unterminated string at byte %d", quote)
}

func skipDollarQuoted(source []byte, offset int) (int, bool, error) {
	end := offset + 1
	if end < len(source) && source[end] == '$' {
		end++
	} else {
		if end >= len(source) || !isDollarTagStart(source[end]) {
			return offset, false, nil
		}
		end++
		for end < len(source) && isDollarTagPart(source[end]) {
			end++
		}
		if end >= len(source) || source[end] != '$' {
			return offset, false, nil
		}
		end++
	}

	delimiter := source[offset:end]
	closing := bytes.Index(source[end:], delimiter)
	if closing < 0 {
		return 0, true, fmt.Errorf("unterminated dollar-quoted string at byte %d", offset)
	}

	return end + closing + len(delimiter), true, nil
}

func scanDelimited(source []byte, offset int, open, close byte) (int, error) {
	for current := offset + 1; current < len(source); current++ {
		if source[current] != close {
			continue
		}
		if open == close && current+1 < len(source) && source[current+1] == close {
			current++
			continue
		}
		return current + 1, nil
	}

	return 0, fmt.Errorf("unterminated quoted identifier at byte %d", offset)
}

func scanWord(source []byte, offset int) int {
	for offset < len(source) {
		r, size := utf8.DecodeRune(source[offset:])
		if r == utf8.RuneError && size == 1 || !isIdentifierPart(r) {
			break
		}
		offset += size
	}
	return offset
}

func isPrefixedString(source []byte, offset int) bool {
	if offset > 0 {
		previous, _ := utf8.DecodeLastRune(source[:offset])
		if isIdentifierPart(previous) {
			return false
		}
	}
	if offset+1 >= len(source) {
		return false
	}

	switch source[offset] {
	case 'b', 'B', 'e', 'E', 'n', 'N', 'x', 'X':
		return source[offset+1] == '\''
	case 'u', 'U':
		return offset+2 < len(source) && source[offset+1] == '&' && source[offset+2] == '\''
	default:
		return false
	}
}

func isUnicodeQuotedIdentifier(source []byte, offset int) bool {
	if offset > 0 {
		previous, _ := utf8.DecodeLastRune(source[:offset])
		if isIdentifierPart(previous) {
			return false
		}
	}
	return offset+2 < len(source) &&
		(source[offset] == 'u' || source[offset] == 'U') &&
		source[offset+1] == '&' && source[offset+2] == '"'
}

func isIdentifierStart(r rune) bool {
	return r == '_' || r >= utf8.RuneSelf || unicode.IsLetter(r)
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || r == '$' || unicode.IsDigit(r)
}

func isSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

func isDollarTagStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isDollarTagPart(value byte) bool {
	return isDollarTagStart(value) || value >= '0' && value <= '9'
}
