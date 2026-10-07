package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppPagesAndData(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "apps/hive.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`../data/hive.js`, `../assets/explorer.js`, `id="main"`} {
		if !strings.Contains(string(page), want) {
			t.Errorf("app page missing %s", want)
		}
	}
	data, _ := os.ReadFile(filepath.Join(out, "data/hive.js"))
	for _, want := range []string{`HB.app = `, `"mods":{`, `"orders":{`, `"cycles":[["basket","orders"]]`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("app data missing %s", want)
		}
	}
	if broken := CheckLinks(out); len(broken) > 0 {
		t.Fatalf("broken: %v", broken)
	}
}
