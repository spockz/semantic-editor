// Package rust contains the small source scanner used by bounded Rust refactorings.
// WHY: Generic type parameters can shadow primitive names inside local items, and their nested bounds require balanced token parsing.
package rust

import (
	"unicode"
	"unicode/utf8"
)

type rustSourceToken struct {
	text       string
	start, end int
}

func rustVisibleGenericTypeParameter(source []byte, position int, name string) bool {
	tokens := rustLexSource(source)
	for i, token := range tokens {
		if token.text != "fn" && token.text != "impl" && token.text != "trait" {
			continue
		}
		genericStart := i + 1
		if token.text != "impl" {
			genericStart = rustAfterIdentifier(tokens, genericStart)
		}
		if genericStart < 0 || genericStart >= len(tokens) || tokens[genericStart].text != "<" {
			continue
		}
		parameters, end, ok := rustGenericParameters(tokens, genericStart, name)
		if !ok || !parameters {
			continue
		}
		bodyStart, bodyEnd, found := rustItemBody(tokens, end+1)
		if found && position >= bodyStart && position < bodyEnd {
			return true
		}
	}
	return false
}

func rustSourceHasGenericTypeParameter(source []byte, name string) bool {
	tokens := rustLexSource(source)
	for i, token := range tokens {
		if token.text != "fn" && token.text != "impl" && token.text != "trait" && token.text != "struct" && token.text != "enum" && token.text != "union" && token.text != "type" {
			continue
		}
		genericStart := i + 1
		if token.text != "impl" {
			genericStart = rustAfterIdentifier(tokens, genericStart)
		}
		if genericStart < 0 || genericStart >= len(tokens) || tokens[genericStart].text != "<" {
			continue
		}
		if parameters, _, ok := rustGenericParameters(tokens, genericStart, name); ok && parameters {
			return true
		}
	}
	return false
}

func rustSourceMayShadowPrimitiveType(source []byte, primitive string) bool {
	tokens := rustLexSource(source)
	for i := range tokens {
		switch tokens[i].text {
		case "type", "struct", "enum", "union", "mod", "trait":
			if rustIdentifierAt(tokens, i+1) == primitive {
				return true
			}
		case "use":
			if rustItemContainsIdentifier(tokens, i+1, primitive) {
				return true
			}
		case "extern":
			if i+1 < len(tokens) && tokens[i+1].text == "crate" && rustItemContainsIdentifier(tokens, i+2, primitive) {
				return true
			}
		}
	}
	return false
}

func rustIdentifierAt(tokens []rustSourceToken, index int) string {
	if index < 0 || index >= len(tokens) {
		return ""
	}
	if tokens[index].text == "r" && index+2 < len(tokens) && tokens[index+1].text == "#" {
		return tokens[index+2].text
	}
	return tokens[index].text
}

func rustItemContainsIdentifier(tokens []rustSourceToken, start int, name string) bool {
	for i := start; i < len(tokens); i++ {
		if tokens[i].text == ";" {
			return false
		}
		if rustIdentifierAt(tokens, i) == name {
			return true
		}
	}
	return false
}

func rustAfterIdentifier(tokens []rustSourceToken, index int) int {
	if index >= len(tokens) || !rustIdentifierToken(tokens[index].text) {
		return -1
	}
	if tokens[index].text == "r" && index+2 < len(tokens) && tokens[index+1].text == "#" && rustIdentifierToken(tokens[index+2].text) {
		return index + 3
	}
	return index + 1
}

func rustGenericParameters(tokens []rustSourceToken, open int, name string) (bool, int, bool) {
	depth := 1
	paren, bracket, brace := 0, 0, 0
	segmentStart := open + 1
	containsName := false
	for i := open + 1; i < len(tokens); i++ {
		switch tokens[i].text {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			bracket++
		case "]":
			bracket--
		case "{":
			brace++
		case "}":
			brace--
		case "<":
			if brace == 0 {
				depth++
			}
		case ">":
			if brace == 0 && depth > 0 && !rustArrowGreater(tokens, i) {
				depth--
			}
			if depth == 0 {
				containsName = containsName || rustGenericParameterHasName(tokens[segmentStart:i], name)
				return containsName, i, true
			}
		case ",":
			if depth == 1 && paren == 0 && bracket == 0 && brace == 0 {
				containsName = containsName || rustGenericParameterHasName(tokens[segmentStart:i], name)
				segmentStart = i + 1
			}
		}
	}
	return containsName, 0, false
}

func rustArrowGreater(tokens []rustSourceToken, index int) bool {
	return index > 0 && tokens[index-1].text == "-" && tokens[index-1].end == tokens[index].start
}

func rustGenericParameterHasName(segment []rustSourceToken, name string) bool {
	if len(segment) == 0 {
		return false
	}
	if segment[0].text == "const" || segment[0].text == "'" {
		return false
	}
	first := 0
	if segment[0].text == "r" && len(segment) >= 3 && segment[1].text == "#" {
		first = 2
	}
	return first < len(segment) && segment[first].text == name
}

func rustItemBody(tokens []rustSourceToken, start int) (int, int, bool) {
	paren, bracket, angle := 0, 0, 0
	for i := start; i < len(tokens); i++ {
		switch tokens[i].text {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			bracket++
		case "]":
			bracket--
		case "<":
			angle++
		case ">":
			if angle > 0 {
				angle--
			}
		case ";":
			if paren == 0 && bracket == 0 && angle == 0 {
				return 0, 0, false
			}
		case "{":
			if paren == 0 && bracket == 0 && angle == 0 {
				end, ok := rustMatchingBrace(tokens, i)
				if ok {
					return tokens[i].end, tokens[end].start, true
				}
				return 0, 0, false
			}
		}
	}
	return 0, 0, false
}

func rustMatchingBrace(tokens []rustSourceToken, open int) (int, bool) {
	depth := 0
	for i := open; i < len(tokens); i++ {
		switch tokens[i].text {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func rustLexSource(source []byte) []rustSourceToken {
	tokens := make([]rustSourceToken, 0, len(source)/3)
	for i := 0; i < len(source); {
		next := rustSkipWhitespaceAndComments(source, i)
		if next != i {
			i = next
			continue
		}
		if end, ok := rustSkipString(source, i); ok {
			i = end
			continue
		}
		token, end := rustLexTokenAt(source, i)
		tokens = append(tokens, token)
		i = end
	}
	return tokens
}

func rustSkipWhitespaceAndComments(source []byte, start int) int {
	if start >= len(source) {
		return start
	}
	switch source[start] {
	case ' ', '\t', '\n', '\r':
		return start + 1
	case '/':
		if start+1 < len(source) {
			switch source[start+1] {
			case '/':
				return rustSkipLineComment(source, start+2)
			case '*':
				return rustSkipBlockComment(source, start+2)
			}
		}
	}
	return start
}

func rustLexTokenAt(source []byte, start int) (rustSourceToken, int) {
	r, size := utf8.DecodeRune(source[start:])
	end := start + size
	if rustIdentifierStart(r) {
		for end < len(source) {
			next, nextSize := utf8.DecodeRune(source[end:])
			if !rustIdentifierContinue(next) {
				break
			}
			end += nextSize
		}
	} else if r >= '0' && r <= '9' {
		for end < len(source) {
			next := rune(source[end])
			if next != '_' && (next < '0' || next > '9') && (next < 'a' || next > 'z') && (next < 'A' || next > 'Z') {
				break
			}
			end++
		}
	}
	return rustSourceToken{text: string(source[start:end]), start: start, end: end}, end
}

func rustSkipLineComment(source []byte, start int) int {
	for start < len(source) && source[start] != '\n' {
		start++
	}
	return start
}

func rustSkipBlockComment(source []byte, start int) int {
	depth := 1
	for start < len(source) && depth > 0 {
		switch {
		case start+1 < len(source) && source[start] == '/' && source[start+1] == '*':
			depth++
			start += 2
		case start+1 < len(source) && source[start] == '*' && source[start+1] == '/':
			depth--
			start += 2
		default:
			start++
		}
	}
	return start
}

func rustSkipString(source []byte, start int) (int, bool) {
	prefix := start
	if source[prefix] == 'b' && prefix+1 < len(source) {
		prefix++
	}
	if end, ok := rustSkipRawString(source, prefix); ok {
		return end, true
	}
	if prefix < len(source) && source[prefix] == '\'' {
		return rustSkipCharLiteral(source, prefix)
	}
	if prefix >= len(source) || source[prefix] != '"' {
		return 0, false
	}
	return rustSkipQuotedString(source, prefix), true
}

func rustSkipRawString(source []byte, prefix int) (int, bool) {
	if prefix >= len(source) || source[prefix] != 'r' {
		return 0, false
	}
	quote := prefix + 1
	for quote < len(source) && source[quote] == '#' {
		quote++
	}
	if quote >= len(source) || source[quote] != '"' {
		return 0, false
	}
	return rustRawStringEnd(source, quote, quote-prefix-1), true
}

func rustRawStringEnd(source []byte, quote, hashes int) int {
	for i := quote + 1; i < len(source); i++ {
		if source[i] != '"' || i+hashes >= len(source) {
			continue
		}
		matches := true
		for j := range hashes {
			matches = matches && source[i+1+j] == '#'
		}
		if matches {
			return i + 1 + hashes
		}
	}
	return len(source)
}

func rustSkipQuotedString(source []byte, quote int) int {
	for i := quote + 1; i < len(source); i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == '"' {
			return i + 1
		}
	}
	return len(source)
}

func rustSkipCharLiteral(source []byte, start int) (int, bool) {
	if start+2 >= len(source) || source[start] != '\'' {
		return 0, false
	}
	if source[start+1] == '\\' {
		i := start + 2
		if i >= len(source) {
			return 0, false
		}
		switch source[i] {
		case 'x':
			i += 3
		case 'u':
			if i+1 >= len(source) || source[i+1] != '{' {
				return 0, false
			}
			i += 2
			for i < len(source) && source[i] != '}' {
				i++
			}
			i++
		default:
			i++
		}
		if i < len(source) && source[i] == '\'' {
			return i + 1, true
		}
		return 0, false
	}
	_, size := utf8.DecodeRune(source[start+1:])
	end := start + 1 + size
	if end < len(source) && source[end] == '\'' {
		return end + 1, true
	}
	return 0, false
}

func rustIdentifierStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func rustIdentifierContinue(r rune) bool {
	return rustIdentifierStart(r) || unicode.IsDigit(r)
}

func rustIdentifierToken(value string) bool {
	if value == "" {
		return false
	}
	first, size := utf8.DecodeRuneInString(value)
	if !rustIdentifierStart(first) {
		return false
	}
	for len(value) > size {
		value = value[size:]
		r, next := utf8.DecodeRuneInString(value)
		if !rustIdentifierContinue(r) {
			return false
		}
		size = next
	}
	return true
}
