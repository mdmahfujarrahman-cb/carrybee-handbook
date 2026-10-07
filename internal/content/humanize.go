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
	return capitalise(strings.Join(words, " "))
}

// HumanText turns a text status such as "on_the_way_to_lmh" into "On the way to lmh".
func HumanText(value string) string {
	return capitalise(strings.ReplaceAll(value, "_", " "))
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
