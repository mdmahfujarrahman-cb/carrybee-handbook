package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "handbook.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadResolvesPathsRelativeToConfig(t *testing.T) {
	p := writeConfig(t, `{"siteTitle":"X","apps":[{"id":"hive","path":"../hive"}]}`)
	c, err := Load(p, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(filepath.Join(filepath.Dir(p), "../hive"))
	if c.Apps[0].Path != want {
		t.Fatalf("path = %q, want %q", c.Apps[0].Path, want)
	}
	if c.Apps[0].Label != "hive" {
		t.Fatalf("label should default to id, got %q", c.Apps[0].Label)
	}
}

func TestLoadAppsRootOverride(t *testing.T) {
	p := writeConfig(t, `{"apps":[{"id":"hive","path":"../hive"}]}`)
	c, err := Load(p, "/ci/repos")
	if err != nil {
		t.Fatal(err)
	}
	if c.Apps[0].Path != "/ci/repos/hive" {
		t.Fatalf("got %q", c.Apps[0].Path)
	}
}

func TestLoadRejectsDuplicateIDs(t *testing.T) {
	p := writeConfig(t, `{"apps":[{"id":"a","path":"x"},{"id":"a","path":"y"}]}`)
	if _, err := Load(p, ""); err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestLoadRejectsEmptyID(t *testing.T) {
	p := writeConfig(t, `{"apps":[{"id":"","path":"x"}]}`)
	if _, err := Load(p, ""); err == nil {
		t.Fatal("expected empty id error")
	}
}

func TestLoadSetsSnapshotPathNextToConfig(t *testing.T) {
	p := writeConfig(t, `{"apps":[{"id":"hive","path":"../hive"}]}`)
	c, err := Load(p, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(p), "snapshot", "scan.json"); c.Snapshot != want {
		t.Fatalf("snapshot = %q, want %q", c.Snapshot, want)
	}
}
