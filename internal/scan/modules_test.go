package scan

import (
	"path/filepath"
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
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic comments",
			in:   "a // line\nb /* block\nstill */ c\nconst u = 'http://x.io';",
			want: "a \nb \n c\nconst u = 'http://x.io';",
		},
		{
			name: "escaped quote",
			in:   "'it\\'s // not'",
			want: "'it\\'s // not'",
		},
		{
			name: "template with expression",
			in:   "`a ${ b /* c */ } // d`",
			want: "`a ${ b  } // d`",
		},
		{
			name: "newline terminates single quote",
			in:   "<p>Don't\n// comment\nx",
			want: "<p>Don't\n\nx",
		},
		{
			name: "newline terminates double quote",
			in:   "\"string\n// comment\nx",
			want: "\"string\n\nx",
		},
		{
			name: "unterminated block comment at EOF",
			in:   "x /* unclosed",
			want: "x ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripComments(tt.in)
			if got != tt.want {
				t.Errorf("StripComments(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
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
	// list.tsx, services/api.ts and types/index.ts (the text status fixture)
	if g.Modules["orders"].Files != 3 {
		t.Fatalf("files = %d", g.Modules["orders"].Files)
	}

	// JSX apostrophe should not open a string that swallows comments
	if g.Modules["basket"].Deps["warehouses"] != 0 {
		t.Fatalf("basket should not depend on warehouses (import is commented): %v", g.Modules["basket"].Deps)
	}

	// Basket should have 2 files (index.ts + view.tsx)
	if g.Modules["basket"].Files != 2 {
		t.Fatalf("basket files = %d, want 2", g.Modules["basket"].Files)
	}

	// Routes: both app files should import modules
	if g.RouteFiles["r"] != 2 {
		t.Fatalf("route file count = %d, want 2", g.RouteFiles["r"])
	}

	// Route imports
	if g.RouteImports["r"]["orders"] != 1 {
		t.Fatalf("route imports orders = %d, want 1", g.RouteImports["r"]["orders"])
	}
	if g.RouteImports["r"]["warehouses"] != 1 {
		t.Fatalf("route imports warehouses = %d, want 1", g.RouteImports["r"]["warehouses"])
	}

	// UsedByRoutes
	if g.Modules["orders"].UsedByRoutes["r"] != 1 {
		t.Fatalf("orders usedByRoutes = %v", g.Modules["orders"].UsedByRoutes)
	}
	if g.Modules["warehouses"].UsedByRoutes["r"] != 1 {
		t.Fatalf("warehouses usedByRoutes = %v", g.Modules["warehouses"].UsedByRoutes)
	}
}
