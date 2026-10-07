package scan

import (
	"path/filepath"
	"strings"
	"testing"

	"carrybee/handbook/internal/config"
)

func fixtureApp(t *testing.T) config.App {
	t.Helper()
	p, err := filepath.Abs("testdata/app")
	if err != nil {
		t.Fatal(err)
	}
	return config.App{ID: "fixture", Path: p}
}

func TestStripComments(t *testing.T) {
	in := "a // line\nb /* block\nstill */ c\nconst u = 'http://x.io';"
	got := StripComments(in)
	if strings.Contains(got, "line") || strings.Contains(got, "block") {
		t.Fatalf("comments kept: %q", got)
	}
	if !strings.Contains(got, "'http://x.io'") {
		t.Fatalf("string with // was damaged: %q", got)
	}
}

func TestBuildGraphModulesAndEdges(t *testing.T) {
	g, err := BuildGraph(fixtureApp(t), func(rel []string) string { return "r" })
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"orders", "basket", "warehouses"} {
		if g.Modules[m] == nil {
			t.Fatalf("module %q missing", m)
		}
	}
	if _, ok := g.Modules["old"]; ok {
		t.Fatal("module from a commented import must not exist")
	}
	if g.Modules["orders"].Deps["basket"] != 1 || g.Modules["orders"].Deps["warehouses"] != 1 {
		t.Fatalf("orders deps = %v", g.Modules["orders"].Deps)
	}
	if g.Modules["orders"].UsedByModules["basket"] != 1 {
		t.Fatalf("usedBy = %v", g.Modules["orders"].UsedByModules)
	}
	if g.Modules["orders"].Shared["lib/api-client"] == 0 {
		t.Fatalf("shared = %v", g.Modules["orders"].Shared)
	}
	if g.Modules["orders"].Files != 2 {
		t.Fatalf("files = %d", g.Modules["orders"].Files)
	}
}
