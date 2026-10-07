package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
)

func fixtureConfig(t *testing.T) *config.Config {
	t.Helper()
	app, err := filepath.Abs("../scan/testdata/app")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// The real content links to all three app maps, so the fixture builds all three from the same test app.
	apps := []config.App{
		{ID: "hive", Label: "hive", Path: app},
		{ID: "aladdin", Label: "aladdin", Path: app},
		{ID: "accounts", Label: "accounts", Path: app},
	}
	return &config.Config{SiteTitle: "Test", Dir: dir, Apps: apps}
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
	err := filepath.WalkDir(out, func(p string, _ os.DirEntry, err error) error {
		if err != nil || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if loc := abs.Find(b); loc != nil {
			t.Errorf("%s has absolute link %s", p, loc)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// fetchCallRE matches a fetch( call and captures the rest of its line.
var fetchCallRE = regexp.MustCompile(`\bfetch\((.*)`)

func TestNoFetchInTheme(t *testing.T) {
	for _, f := range []string{"../../theme/assets/handbook.js", "../../theme/assets/explorer.js", "../../theme/assets/terms.js"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if m := fetchCallRE.FindStringSubmatch(line); m != nil && !strings.Contains(m[1], "api/") {
				t.Errorf("%s:%d fetch outside api/ (data must come from data/*.js so file:// works): %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// readSearch loads data/search.js from a built site.
func readSearch(t *testing.T, out string) []SearchEntry {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(out, "data/search.js"))
	if err != nil {
		t.Fatal(err)
	}
	_, js, ok := strings.Cut(string(b), "HB.search = ")
	if !ok {
		t.Fatalf("data/search.js has no HB.search: %.80s", b)
	}
	var entries []SearchEntry
	if err := json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimSpace(js), ";")), &entries); err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestSearchIndexCoversEverything(t *testing.T) {
	out := t.TempDir()
	res, err := Build(fixtureConfig(t), out)
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, e := range readSearch(t, out) {
		have[e.Type+" "+e.Title+" "+e.Sub+" "+e.Href] = true
		have[e.Type+" "+e.Href] = true
	}
	want := func(key string) {
		if !have[key] {
			t.Errorf("search index missing %q", key)
		}
	}
	data, err := content.LoadData("../../data")
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range data.Terms {
		want("term " + termAnchor(term.ID))
	}
	for app, sc := range res.Scans {
		for m := range sc.Modules {
			want("module " + moduleAnchor(app, m))
		}
		for r := range sc.Routes {
			want("route " + r + " " + app + " page " + appOut(app))
		}
		for d := range sc.APIDomains {
			want("api " + apiIndexOut + "#api." + app + "." + d)
		}
		for _, st := range sc.Statuses {
			id := strconv.Itoa(st.ID)
			want("status " + id + " " + content.HumanStatus(st.Name) + " order status " + statusesOut + "#s." + id)
		}
		for _, v := range sc.TextStatuses {
			want("status " + content.HumanText(v) + " " + app + " text status " + statusesOut + "#text-statuses")
		}
	}
	if len(res.Scans) == 0 {
		t.Fatal("fixture scanned no apps")
	}
}

func TestSkippedFilesBecomeWarnings(t *testing.T) {
	s := &site{data: &content.Data{}, res: &Result{}, scans: map[string]*scan.AppScan{
		"hive": {Modules: map[string]*scan.Module{}, Skipped: []string{"modules/a/x.ts"}},
	}}
	s.collectMissing()
	if want := "hive: skipped unreadable modules/a/x.ts"; len(s.res.Warnings) != 1 || s.res.Warnings[0] != want {
		t.Fatalf("warnings = %v, want [%s]", s.res.Warnings, want)
	}
}

func TestCheckLinksVerifiesFragments(t *testing.T) {
	dir := t.TempDir()
	pages := map[string]string{
		"index.html": `<main id="content"><a href="#content">skip</a><a href="#nowhere">x</a>` +
			`<a href="b.html#here">ok</a><a href="b.html#gone">x</a><a href="b.html#t.cod">js</a><a href="b.html#mod.orders">js</a></main>`,
		"b.html": `<h2 id="here">Here</h2>`,
	}
	for name, body := range pages {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := CheckLinks(dir)
	want := []string{"index.html -> #nowhere", "index.html -> b.html#gone"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CheckLinks = %v, want %v", got, want)
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

func TestMissingNamesTheFileToEdit(t *testing.T) {
	s := &site{res: &Result{},
		data: &content.Data{Modules: map[string]map[string]content.Desc{}, Groups: map[string]map[string]content.Desc{
			"hive": {"top level": {}},
		}},
		scans: map[string]*scan.AppScan{"hive": {
			Modules: map[string]*scan.Module{"orders": {}},
			Routes:  map[string]*scan.Route{"(oms)/orders": {}, "dashboard": {}},
		}},
		terms: []content.Term{{ID: "cod"}},
	}
	s.collectMissing()
	want := []string{
		"page group hive/oms has no description in data/groups.json",
		"module hive/orders has no description in data/modules.json",
		"term cod is not linked to any module (add a pin in data/term-pins.json)",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(s.res.Missing, want) {
		t.Fatalf("missing = %q, want %q", s.res.Missing, want)
	}
}
