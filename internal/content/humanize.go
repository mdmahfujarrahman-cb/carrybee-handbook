package content

import (
	"strings"
	"unicode"
)

const statusPrefix = "TransferStatus"

// HumanStatus turns an enum member name such as "TransferStatusAtSubSort"
// into readable text: "At sub sort".
func HumanStatus(name string) string {
	if rest := strings.TrimPrefix(name, statusPrefix); rest != "" {
		name = rest
	}
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = nil
		}
	}
	for _, r := range name {
		if unicode.IsUpper(r) {
			flush()
		}
		cur = append(cur, unicode.ToLower(r))
	}
	flush()
	text := strings.Join(words, " ")
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + text[1:]
}
