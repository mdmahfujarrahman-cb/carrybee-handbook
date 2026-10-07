package content

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	kindTerm   = "term"
	kindModule = "module"
	kindStatus = "status"
	fmFence    = "---\n"
)

type Ref struct{ Kind, App, ID string }

type Page struct {
	Path, Section, Title, Level, Summary string
	Order                                int
	// Roles is reserved for role filtering; it is parsed from front matter but not used yet.
	Roles []string
	Body  string
	Refs  []Ref
}

var shortcodeRE = regexp.MustCompile(`\{\{\s*(term|module|status)\s+([^}]*)\}\}`)
var argRE = regexp.MustCompile(`"([^"]*)"|(\d+)`)

func shortcodeArgs(raw string) []string {
	var out []string
	for _, m := range argRE.FindAllStringSubmatch(raw, -1) {
		if m[1] != "" {
			out = append(out, m[1])
		} else {
			out = append(out, m[2])
		}
	}
	return out
}

func parseFrontMatter(src string) (map[string]string, string) {
	meta := map[string]string{}
	if !strings.HasPrefix(src, fmFence) {
		return meta, src
	}
	rest := src[len(fmFence):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return meta, src
	}
	sc := bufio.NewScanner(strings.NewReader(rest[:end]))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), ":"); ok {
			meta[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	body := strings.TrimPrefix(rest[end+len("\n---"):], "\n")
	return meta, body
}

func splitRoles(s string) []string {
	var roles []string
	for _, r := range strings.Split(s, ",") {
		if r = strings.TrimSpace(r); r != "" {
			roles = append(roles, r)
		}
	}
	return roles
}

func extractRefs(body string) []Ref {
	var refs []Ref
	for _, m := range shortcodeRE.FindAllStringSubmatch(body, -1) {
		args := shortcodeArgs(m[2])
		switch {
		case m[1] == kindModule && len(args) == 2:
			refs = append(refs, Ref{Kind: kindModule, App: args[0], ID: args[1]})
		case m[1] != kindModule && len(args) >= 1:
			refs = append(refs, Ref{Kind: m[1], ID: args[0]})
		}
	}
	return refs
}

func readPage(dir, p string) (Page, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return Page{}, err
	}
	rel, err := filepath.Rel(dir, p)
	if err != nil {
		return Page{}, err
	}
	meta, body := parseFrontMatter(string(raw))
	page := Page{Path: strings.TrimSuffix(filepath.ToSlash(rel), ".md"), Body: body,
		Title: meta["title"], Section: meta["section"], Level: meta["level"], Summary: meta["summary"],
		Roles: splitRoles(meta["roles"]), Refs: extractRefs(body)}
	if o := meta["order"]; o != "" {
		if page.Order, err = strconv.Atoi(o); err != nil {
			return Page{}, &os.PathError{Op: "order", Path: p, Err: err}
		}
	}
	return page, nil
}

func sortPages(pages []Page) {
	sort.SliceStable(pages, func(i, j int) bool {
		a, b := pages[i], pages[j]
		if a.Section != b.Section {
			return a.Section < b.Section
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.Title < b.Title
	})
}

func LoadPages(dir string) ([]Page, error) {
	var pages []Page
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		page, err := readPage(dir, p)
		if err != nil {
			return err
		}
		pages = append(pages, page)
		return nil
	})
	sortPages(pages)
	return pages, err
}
