// Package theme embeds the handbook templates and static assets.
package theme

import "embed"

// FS holds templates/*.html, assets/* and diagrams/*.html (inline SVG figures used by {{diagram}}).
//
//go:embed templates/*.html assets/* diagrams/*.html
var FS embed.FS
