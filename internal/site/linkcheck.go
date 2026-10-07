package site

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	linkRE = regexp.MustCompile(`(?:href|src)="([^"]*)"`)
	idRE   = regexp.MustCompile(`\sid="([^"]+)"`)
)

// jsAnchorPrefixes are fragments whose targets are rendered by JavaScript (terms list, app maps),
// so they are not present as ids in the built HTML.
var jsAnchorPrefixes = []string{"t.", "mod."}

// skipTarget reports links that are external or template expressions inside inline JS (Ruling R3).
func skipTarget(target string) bool {
	return strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") ||
		strings.HasPrefix(target, "data:") || strings.Contains(target, "${") || strings.Contains(target, "{{")
}

func jsAnchor(frag string) bool {
	for _, p := range jsAnchorPrefixes {
		if strings.HasPrefix(frag, p) {
			return true
		}
	}
	return false
}

// linkChecker checks the links of every page under dir, caching the ids of each target page.
type linkChecker struct {
	dir string
	ids map[string]map[string]bool
}

// idsOf returns the element ids in the HTML file at p.
func (c *linkChecker) idsOf(p string) (map[string]bool, error) {
	if ids, ok := c.ids[p]; ok {
		return ids, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, m := range idRE.FindAllSubmatch(b, -1) {
		ids[string(m[1])] = true
	}
	c.ids[p] = ids
	return ids, nil
}

// broken reports whether the link target (relative to page p) is missing, or is an HTML page
// without the id named by its #fragment.
func (c *linkChecker) broken(p, target string) bool {
	target, _, _ = strings.Cut(target, "?")
	file, frag, _ := strings.Cut(target, "#")
	dest := p
	if file != "" {
		dest = filepath.Join(filepath.Dir(p), file)
	}
	if _, err := os.Stat(dest); err != nil {
		return true
	}
	if frag == "" || jsAnchor(frag) || !strings.HasSuffix(dest, ".html") {
		return false
	}
	ids, err := c.idsOf(dest)
	return err != nil || !ids[frag]
}

func (c *linkChecker) brokenIn(p string, b []byte) []string {
	var broken []string
	for _, m := range linkRE.FindAllStringSubmatch(string(b), -1) {
		target := m[1]
		if target == "" || target == "#" || skipTarget(target) || !c.broken(p, target) {
			continue
		}
		rel, err := filepath.Rel(c.dir, p)
		if err != nil {
			rel = p
		}
		broken = append(broken, filepath.ToSlash(rel)+" -> "+target)
	}
	return broken
}

// CheckLinks returns "page -> target" for relative links that point at missing files, or at
// HTML pages without the #fragment id. Unreadable files are reported as problems too.
func CheckLinks(dir string) []string {
	c := &linkChecker{dir: dir, ids: map[string]map[string]bool{}}
	var broken []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		broken = append(broken, c.brokenIn(p, b)...)
		return nil
	})
	if err != nil {
		broken = append(broken, "walk "+dir+": "+err.Error())
	}
	return broken
}
