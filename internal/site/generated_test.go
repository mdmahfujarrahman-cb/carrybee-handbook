package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedPages(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join(out, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if api := read("code/api-index.html"); !strings.Contains(api, `id="api.hive.orders"`) || !strings.Contains(api, "(oms)/orders") {
		t.Errorf("api index missing orders row with its page")
	}
	if st := read("business/statuses.html"); !strings.Contains(st, `id="s.15"`) || !strings.Contains(st, "Delivered") {
		t.Errorf("statuses table missing 15 Delivered")
	}
	if terms := read("terms.html"); !strings.Contains(terms, "data/terms.js") {
		t.Errorf("terms page must load data/terms.js")
	}
}

func TestStatusPhase(t *testing.T) {
	cases := map[int]string{2: "First mile", 9: "Middle mile", 14: "Last mile", 15: "Final", 25: "Return",
		43: "Reverse pickup", 38: "Middle mile", 53: "Change request"}
	for id, want := range cases {
		if got := statusPhase(id); got != want {
			t.Errorf("statusPhase(%d) = %q want %q", id, got, want)
		}
	}
}
