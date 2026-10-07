package content

import (
	"strings"
	"testing"
)

func testResolver() Resolver {
	return Resolver{
		TermName: func(id string) (string, bool) {
			return map[string]string{"cash-in-hand": "Cash in hand"}[id], id == "cash-in-hand"
		},
		ModuleDesc: func(app, id string) (Desc, bool) {
			return Desc{"Hub ledger", "Cash book"}, app == "hive" && id == "hub-ledger"
		},
		StatusName: func(id int) (string, bool) { return "Delivered", id == 15 },
		ModuleHref: func(app, id string) string { return "../apps/" + app + ".html#mod." + id },
	}
}

func TestLoadPagesFrontMatter(t *testing.T) {
	pages, err := LoadPages("testdata/content")
	if err != nil {
		t.Fatal(err)
	}
	p := pages[0]
	if p.Path != "business/sample" || p.Title != "Sample" || p.Order != 2 || p.Level != "basics" {
		t.Fatalf("%+v", p)
	}
	if strings.Join(p.Roles, ",") != "frontend,qa" {
		t.Fatalf("roles = %v", p.Roles)
	}
	if len(p.Refs) != 3 {
		t.Fatalf("refs = %v", p.Refs)
	}
}

func TestRenderShortcodesAndCallouts(t *testing.T) {
	pages, _ := LoadPages("testdata/content")
	html, err := Render(pages[0].Body, testResolver())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<span class="term" data-term="cash-in-hand">Cash in hand</span>`,
		`href="../apps/hive.html#mod.hub-ledger"`,
		`<span class="pill st ok">15 Delivered</span>`,
		`<aside class="callout carrybee">`,
		`<strong>label</strong>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s in\n%s", want, html)
		}
	}
}

func TestCheckUnknownReference(t *testing.T) {
	pages := []Page{{Path: "x/y", Refs: []Ref{{Kind: "term", ID: "nope"}, {Kind: "module", App: "hive", ID: "ghost"}, {Kind: "status", ID: "99"}}}}
	problems := Check(pages, testResolver())
	if len(problems) != 3 {
		t.Fatalf("problems = %v", problems)
	}
	if !strings.Contains(problems[0], "x/y") || !strings.Contains(problems[0], "nope") {
		t.Fatalf("problem must name file and reference: %q", problems[0])
	}
}

func TestTermLabelOverride(t *testing.T) {
	src := `Into a {{term "cash-in-hand" "cash in hand"}} and {{term "cash-in-hand"}}.`
	html, err := Render(src, testResolver())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `<span class="term" data-term="cash-in-hand">cash in hand</span>`) {
		t.Errorf("label not applied: %s", html)
	}
	if !strings.Contains(html, `data-term="cash-in-hand">Cash in hand</span>`) {
		t.Errorf("default name lost: %s", html)
	}
	refs := extractRefs(src)
	if len(refs) != 2 || refs[0] != (Ref{Kind: "term", ID: "cash-in-hand"}) {
		t.Errorf("refs = %v", refs)
	}
}

func TestHeadingsGetIDs(t *testing.T) {
	html, err := Render("## Money flow\n\ntext", testResolver())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `<h2 id="money-flow">Money flow</h2>`) {
		t.Fatalf("heading has no id: %s", html)
	}
}
