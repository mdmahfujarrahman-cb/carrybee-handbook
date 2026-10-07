package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"carrybee/handbook/internal/config"
)

func fixtureConfig(t *testing.T) *config.Config {
	t.Helper()
	app, _ := filepath.Abs("../scan/testdata/app")
	dir, _ := filepath.Abs("../..")
	return &config.Config{SiteTitle: "Test", Dir: dir, Apps: []config.App{{ID: "hive", Label: "hive", Path: app}}}
}

func TestRel(t *testing.T) {
	cases := [][3]string{
		{"index.html", "terms.html#t.x", "terms.html#t.x"},
		{"business/courier-101.html", "terms.html", "../terms.html"},
		{"business/courier-101.html", "business/statuses.html", "statuses.html"},
		{"apps/hive.html", "assets/handbook.css", "../assets/handbook.css"},
	}
	for _, c := range cases {
		if got := Rel(c[0], c[1]); got != c[2] {
			t.Errorf("Rel(%q,%q) = %q want %q", c[0], c[1], got, c[2])
		}
	}
}

func TestBuildProducesSiteWithValidLinks(t *testing.T) {
	out := t.TempDir()
	res, err := Build(fixtureConfig(t), out)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"index.html", "terms.html", "assets/handbook.css", "assets/handbook.js", "data/search.js", "data/sync.js"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if broken := CheckLinks(out); len(broken) > 0 {
		t.Fatalf("broken links: %v", broken)
	}
	if res.Pages == 0 {
		t.Fatal("no content pages rendered")
	}
}

func TestLinksAreRelative(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	abs := regexp.MustCompile(`(?:href|src)="/[^/]`)
	filepath.WalkDir(out, func(p string, d os.DirEntry, err error) error {
		if strings.HasSuffix(p, ".html") {
			b, _ := os.ReadFile(p)
			if loc := abs.Find(b); loc != nil {
				t.Errorf("%s has absolute link %s", p, loc)
			}
		}
		return nil
	})
}

func TestNoFetchInTheme(t *testing.T) {
	for _, f := range []string{"../../theme/assets/handbook.js", "../../theme/assets/explorer.js"} {
		b, err := os.ReadFile(f)
		if err != nil {
			continue // explorer.js arrives in Task 7
		}
		if strings.Contains(string(b), "fetch(") && !strings.Contains(string(b), "/api/") {
			t.Errorf("%s uses fetch for data; data must come from data/*.js so file:// works", f)
		}
	}
}

func TestSearchIndexCoversEverything(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "data/search.js"))
	for _, want := range []string{`"type":"module"`, `"title":"orders"`, `"type":"term"`, `"type":"status"`, `"type":"api"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("search index missing %s", want)
		}
	}
}

// Ruling R3: template expressions inside inline JS are not links.
func TestCheckLinksSkipsTemplateExpressions(t *testing.T) {
	dir := t.TempDir()
	page := `<a href="ok.html">ok</a><script>x='<a href="${u}">'; y='<img src="{{.X}}">'</script><a href="gone.html">x</a>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.html"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got := CheckLinks(dir)
	if len(got) != 1 || got[0] != "index.html -> gone.html" {
		t.Fatalf("CheckLinks = %v, want only gone.html", got)
	}
}
