// Package scan reads the Carrybee Next.js frontends into structured data.
package scan

import "strings"

// StripComments removes // and /* */ comments while leaving string and template literals intact.
func StripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	var quote byte
	for i := 0; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			b.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				i++
				b.WriteByte(src[i])
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch {
		case c == '\'' || c == '"' || c == '`':
			quote = c
			b.WriteByte(c)
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				b.WriteByte('\n')
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			b.WriteString(strings.Repeat("\n", strings.Count(src[i:i+2+end], "\n")))
			i += end + 3
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
