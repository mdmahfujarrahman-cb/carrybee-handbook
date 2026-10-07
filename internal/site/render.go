package site

import (
	"bytes"
	"html/template"
	"path"
	"strings"

	"carrybee/handbook/theme"
)

// Rel returns a relative URL from the page at `from` to the output path `to` (which may carry #fragment).
func Rel(from, to string) string {
	target, frag, _ := strings.Cut(to, "#")
	fromDir := path.Dir(from)
	if fromDir == "." {
		fromDir = ""
	}
	up := ""
	if fromDir != "" {
		up = strings.Repeat("../", strings.Count(fromDir, "/")+1)
	}
	rel := up + target
	if fromDir != "" && strings.HasPrefix(target, fromDir+"/") {
		rel = strings.TrimPrefix(target, fromDir+"/")
	}
	if frag != "" {
		rel += "#" + frag
	}
	return rel
}

// NavItem is one link in the sidebar, breadcrumb or prev/next row.
type NavItem struct {
	Title, Href string
	Current     bool
}

// NavSection is a titled group of sidebar links.
type NavSection struct {
	Title string
	Items []NavItem
}

// PageView is the data every page template receives.
type PageView struct {
	SiteTitle  string
	Title      string
	Section    string
	Level      string
	Summary    string
	Body       template.HTML
	Nav        []NavSection
	Crumbs     []NavItem
	Prev, Next *NavItem
	Root       string // relative prefix to site root, e.g. "../"
	Wide       bool   // maps and tables use full width
	Extra      any    // template-specific data
}

var levelNames = map[string]string{"basics": "Basics", "carrybee": "Carrybee-specific", "code": "Code"}

var audienceNames = map[string]string{"ops": "Ops panel", "merchant": "Merchant panel", "finance": "Finance panel"}

var audienceWho = map[string]string{
	"ops":      "Used by Carrybee's own operations staff: hub teams, dispatch, support and admins run the delivery network from it.",
	"merchant": "Used by merchants (sellers): they book parcels, track them, raise requests and see what they are paid.",
	"finance":  "Used by Carrybee's finance and legal team: invoices, settlements, payouts and reports.",
}

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"lower":        strings.ToLower,
	"levelName":    func(l string) string { return levelNames[l] },
	"audienceName": func(a string) string { return audienceNames[a] },
	"audienceWho":  func(a string) string { return audienceWho[a] },
	"initial": func(s string) string {
		if s == "" {
			return ""
		}
		return strings.ToUpper(s[:1])
	},
}).ParseFS(theme.FS, "templates/*.html"))

func renderTemplate(name string, v PageView) ([]byte, error) {
	var b bytes.Buffer
	err := templates.ExecuteTemplate(&b, name, v)
	return b.Bytes(), err
}

func rootFor(out string) string {
	n := strings.Count(out, "/")
	return strings.Repeat("../", n)
}
