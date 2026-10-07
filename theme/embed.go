// Package theme embeds the handbook templates and static assets.
package theme

import "embed"

// FS holds templates/*.html and assets/*.
//
//go:embed templates/*.html assets/*
var FS embed.FS
