package site

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
	"carrybee/handbook/theme"
)

// Result summarises a build for the CLI and the Sync server.
type Result struct {
	Scans    map[string]*scan.AppScan
	Pages    int
	Warnings []string
	Missing  []string
	// TermLinks maps term id -> app -> linked modules, for the Sync diff.
	TermLinks map[string]map[string][]string
}

const (
	sectionCode  = "code"
	sectionRoles = "roles"
	homeOut      = "index.html"
)

var sectionOrder = []struct{ ID, Title string }{
	{"business", "Business"}, {sectionCode, "Code"}, {sectionRoles, "Roles"},
}

var sectionTitles = map[string]string{
	"business": "Business", "apps": "Apps", sectionCode: "Code", sectionRoles: "Roles", "": "Reference",
}

type site struct {
	cfg   *config.Config
	out   string
	data  *content.Data
	pages []content.Page
	terms []content.Term
	scans map[string]*scan.AppScan
	res   *Result
}

// Build scans the apps, renders every page and writes a complete site into outDir.
func Build(cfg *config.Config, outDir string) (*Result, error) {
	data, err := content.LoadData(filepath.Join(cfg.Dir, "data"))
	if err != nil {
		return nil, err
	}
	pages, err := content.LoadPages(filepath.Join(cfg.Dir, "content"))
	if err != nil {
		return nil, err
	}
	scans := scan.ScanAll(cfg)
	s := &site{cfg: cfg, out: outDir, data: data, pages: pages, scans: scans,
		res: &Result{Scans: scans}}
	s.terms = LinkTerms(data.Terms, scans, data.Pins)
	s.res.TermLinks = termLinks(s.terms)
	for _, step := range []func() error{s.copyAssets, s.writeData, s.writePages, s.writeTerms,
		s.writeApps, s.writeGenerated} {
		if err := step(); err != nil {
			return s.res, err
		}
	}
	s.collectMissing()
	return s.res, nil
}

func (s *site) write(rel string, b []byte) error {
	p := filepath.Join(s.out, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func (s *site) copyAssets() error {
	return fs.WalkDir(theme.FS, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := theme.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return s.write(p, b)
	})
}

func (s *site) resolver(from string) content.Resolver {
	termNames := map[string]string{}
	for _, t := range s.terms {
		termNames[t.ID] = t.Name
	}
	statuses := map[int]string{}
	for _, sc := range s.scans {
		for _, st := range sc.Statuses {
			statuses[st.ID] = content.HumanStatus(st.Name)
		}
	}
	return content.Resolver{
		TermName: func(id string) (string, bool) { n, ok := termNames[id]; return n, ok },
		ModuleDesc: func(app, id string) (content.Desc, bool) {
			if sc := s.scans[app]; sc == nil || sc.Modules[id] == nil {
				return content.Desc{}, false
			}
			return s.data.Modules[app][id], true
		},
		StatusName: func(id int) (string, bool) { n, ok := statuses[id]; return n, ok },
		ModuleHref: func(app, id string) string { return Rel(from, "apps/"+app+".html#mod."+id) },
		TermHref:   func(id string) string { return Rel(from, "terms.html#t."+id) },
	}
}

type syncApp struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Commit    string `json:"commit"`
	ScannedAt string `json:"scannedAt"`
	Modules   int    `json:"modules"`
	Pages     int    `json:"pages"`
	Error     string `json:"error,omitempty"`
}

func (s *site) syncData() map[string]any {
	commits := map[string]string{}
	apps := []syncApp{}
	for _, a := range s.cfg.Apps {
		sc := s.scans[a.ID]
		if sc == nil {
			continue
		}
		commits[a.ID] = sc.Commit
		apps = append(apps, syncApp{a.ID, a.Label, sc.Commit, sc.ScannedAt.UTC().Format(time.RFC3339),
			len(sc.Modules), len(sc.Routes), sc.Error})
	}
	return map[string]any{"builtAt": time.Now().UTC().Format(time.RFC3339), "commits": commits, "apps": apps}
}

func (s *site) writeData() error {
	files := []struct {
		name, key string
		v         any
	}{
		{"data/search.js", "search", searchIndex(s.pages, s.terms, s.data, s.scans)},
		{"data/terms.js", "terms", map[string]any{"categories": s.data.Categories, "terms": s.terms, "modules": s.data.Modules}},
		{"data/sync.js", "sync", s.syncData()},
	}
	for _, f := range files {
		b, err := jsData(f.key, f.v)
		if err != nil {
			return err
		}
		if err := s.write(f.name, b); err != nil {
			return err
		}
	}
	return nil
}

func (s *site) sectionNav(cur, id, title string) NavSection {
	sec := NavSection{Title: title}
	for _, p := range s.pages {
		if p.Section == id {
			out := pageOut(p)
			sec.Items = append(sec.Items, NavItem{p.Title, Rel(cur, out), out == cur})
		}
	}
	if id == sectionCode {
		sec.Items = append(sec.Items, NavItem{"API domain index", Rel(cur, "code/api-index.html"), cur == "code/api-index.html"})
	}
	return sec
}

// nav builds the sidebar for the page at output path `cur`.
func (s *site) nav(cur string) []NavSection {
	secs := []NavSection{{Title: "Start", Items: []NavItem{{Title: "Home", Href: Rel(cur, homeOut), Current: cur == homeOut}}}}
	for _, so := range sectionOrder {
		secs = append(secs, s.sectionNav(cur, so.ID, so.Title))
	}
	apps := NavSection{Title: "Apps"}
	for _, a := range s.cfg.Apps {
		out := "apps/" + a.ID + ".html"
		apps.Items = append(apps.Items, NavItem{a.Label, Rel(cur, out), out == cur})
	}
	secs = append(secs[:2], append([]NavSection{apps}, secs[2:]...)...)
	secs = append(secs, NavSection{Title: "Reference", Items: []NavItem{
		{"Courier terms", Rel(cur, "terms.html"), cur == "terms.html"},
		{"Known issues", Rel(cur, "known-issues.html"), cur == "known-issues.html"},
	}})
	return secs
}

// crumbs returns Home plus the section of `cur`, linked to that section's first page.
func (s *site) crumbs(cur string, nav []NavSection) []NavItem {
	if cur == homeOut {
		return nil
	}
	out := []NavItem{{Title: "Home", Href: Rel(cur, homeOut)}}
	dir, _, found := strings.Cut(cur, "/")
	if !found {
		dir = ""
	}
	title := sectionTitles[dir]
	for _, sec := range nav {
		if sec.Title == title && len(sec.Items) > 0 && dir != "" {
			return append(out, NavItem{Title: title, Href: sec.Items[0].Href})
		}
	}
	return append(out, NavItem{Title: title})
}

func (s *site) view(cur, title string) PageView {
	nav := s.nav(cur)
	return PageView{SiteTitle: s.cfg.SiteTitle, Title: title, Nav: nav, Crumbs: s.crumbs(cur, nav), Root: rootFor(cur)}
}

func prevNext(v *PageView, p content.Page, siblings []content.Page) {
	out := pageOut(p)
	for i, sib := range siblings {
		if sib.Path != p.Path {
			continue
		}
		if i > 0 {
			v.Prev = &NavItem{Title: siblings[i-1].Title, Href: Rel(out, pageOut(siblings[i-1]))}
		}
		if i+1 < len(siblings) {
			v.Next = &NavItem{Title: siblings[i+1].Title, Href: Rel(out, pageOut(siblings[i+1]))}
		}
	}
}

func (s *site) writePage(p content.Page, siblings []content.Page) error {
	out := pageOut(p)
	body, err := content.Render(p.Body, s.resolver(out))
	if err != nil {
		return fmt.Errorf("render %s: %w", p.Path, err)
	}
	v := s.view(out, p.Title)
	v.Section, v.Level, v.Summary, v.Body = p.Section, p.Level, p.Summary, template.HTML(body)
	prevNext(&v, p, siblings)
	tpl := "article.html"
	switch {
	case out == homeOut:
		tpl = "home.html"
		v.Wide = true
		v.Extra = s.homeExtra(out)
	case p.Section == sectionRoles:
		tpl = "role.html"
	}
	b, err := renderTemplate(tpl, v)
	if err != nil {
		return fmt.Errorf("template %s for %s: %w", tpl, p.Path, err)
	}
	return s.write(out, b)
}

func (s *site) writePages() error {
	bySection := map[string][]content.Page{}
	for _, p := range s.pages {
		bySection[p.Section] = append(bySection[p.Section], p)
	}
	for _, p := range s.pages {
		if err := s.writePage(p, bySection[p.Section]); err != nil {
			return err
		}
		s.res.Pages++
	}
	if problems := content.Check(s.pages, s.resolver(homeOut)); len(problems) > 0 {
		s.res.Warnings = append(s.res.Warnings, problems...)
	}
	return nil
}

// RoleCard is one "pick your role" card on the home page.
type RoleCard struct{ ID, Title, Blurb, Href string }

// AppStat is one app's scan size shown on the home page.
type AppStat struct {
	ID, Label, Audience, Href string
	Modules, Pages            int
	Error                     string
}

func (s *site) homeExtra(cur string) map[string]any {
	var roles []RoleCard
	for _, p := range s.pages {
		if p.Section == sectionRoles {
			roles = append(roles, RoleCard{path.Base(p.Path), p.Title, p.Summary, Rel(cur, pageOut(p))})
		}
	}
	var apps []AppStat
	for _, a := range s.cfg.Apps {
		if sc := s.scans[a.ID]; sc != nil {
			apps = append(apps, AppStat{a.ID, a.Label, a.Audience, Rel(cur, "apps/"+a.ID+".html"),
				len(sc.Modules), len(sc.Routes), sc.Error})
		}
	}
	return map[string]any{"Roles": roles, "Apps": apps, "Terms": len(s.terms),
		"TermsHref":  Rel(cur, "terms.html"),
		"Courier101": Rel(cur, "business/courier-101.html"), "HowCarrybee": Rel(cur, "business/how-carrybee-works.html")}
}

func (s *site) collectMissing() {
	for _, id := range sortedKeys(s.scans) {
		for m := range s.scans[id].Modules {
			if _, ok := s.data.Modules[id][m]; !ok {
				s.res.Missing = append(s.res.Missing, "module "+id+"/"+m+" has no description in data/modules.json")
			}
		}
		if s.scans[id].Error != "" {
			s.res.Warnings = append(s.res.Warnings, id+": not scanned: "+s.scans[id].Error)
		}
	}
	for _, t := range s.terms {
		if len(t.Where) == 0 {
			s.res.Missing = append(s.res.Missing, "term "+t.ID+" is not linked to any module (add a pin)")
		}
	}
	sort.Strings(s.res.Missing)
}

func termLinks(terms []content.Term) map[string]map[string][]string {
	out := make(map[string]map[string][]string, len(terms))
	for _, t := range terms {
		if len(t.Where) > 0 {
			out[t.ID] = t.Where
		}
	}
	return out
}
