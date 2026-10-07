// Package scan reads the Carrybee Next.js frontends into structured data.
package scan

import "strings"

// StripComments removes // and /* */ comments while leaving string and template literals intact.
// Single and double quotes terminate at newline to prevent stray quotes from swallowing code.
// Template literals support ${ } expressions where comments are removed.
// Known limitation: regex literals (e.g., /\/\// or /'//) may cause quote-state desynchronization.
func StripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	var quote byte
	var templateDepth int
	for i := 0; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			if c == '\\' && i+1 < len(src) {
				b.WriteByte(c)
				i++
				b.WriteByte(src[i])
				continue
			}
			if quote == '`' {
				// Template literal: track ${ } expressions
				if c == '$' && i+1 < len(src) && src[i+1] == '{' {
					b.WriteByte(c)
					b.WriteByte('{')
					templateDepth++
					i++
				} else if c == '}' && templateDepth > 0 {
					b.WriteByte(c)
					templateDepth--
				} else if c == '`' && templateDepth == 0 {
					b.WriteByte(c)
					quote = 0
				} else if templateDepth > 0 && c == '/' && i+1 < len(src) && src[i+1] == '/' {
					// Line comment inside expression: remove it
					for i < len(src) && src[i] != '\n' {
						i++
					}
					if i < len(src) {
						b.WriteByte('\n')
					}
				} else if templateDepth > 0 && c == '/' && i+1 < len(src) && src[i+1] == '*' {
					// Block comment inside expression: remove it
					end := strings.Index(src[i+2:], "*/")
					if end < 0 {
						return b.String()
					}
					b.WriteString(strings.Repeat("\n", strings.Count(src[i:i+2+end], "\n")))
					i += end + 3
				} else {
					b.WriteByte(c)
				}
			} else if quote == '\'' || quote == '"' {
				b.WriteByte(c)
				if c == quote {
					quote = 0
				} else if c == '\n' {
					// Single/double quotes terminate at newline
					quote = 0
				}
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
