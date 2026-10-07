package site

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var linkRE = regexp.MustCompile(`(?:href|src)="([^"#?]+)`)

// skipTarget reports links that are external or template expressions inside inline JS (Ruling R3).
func skipTarget(target string) bool {
	return strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") ||
		strings.HasPrefix(target, "data:") || strings.Contains(target, "${") || strings.Contains(target, "{{")
}

func brokenIn(dir, p string, b []byte) []string {
	var broken []string
	for _, m := range linkRE.FindAllStringSubmatch(string(b), -1) {
		target := m[1]
		if skipTarget(target) {
			continue
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(p), target)); err != nil {
			rel, _ := filepath.Rel(dir, p)
			broken = append(broken, filepath.ToSlash(rel)+" -> "+target)
		}
	}
	return broken
}

// CheckLinks returns "page -> target" for relative links that point at missing files.
// Unreadable files are reported as problems too.
func CheckLinks(dir string) []string {
	var broken []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		broken = append(broken, brokenIn(dir, p, b)...)
		return nil
	})
	if err != nil {
		broken = append(broken, "walk "+dir+": "+err.Error())
	}
	return broken
}
