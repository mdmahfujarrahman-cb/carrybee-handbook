package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"carrybee/handbook/internal/config"
)

// snapshotConfig is the fixture config with its snapshot in a temp dir, so tests never touch the repo's snapshot.
func snapshotConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := fixtureConfig(t)
	cfg.Snapshot = filepath.Join(t.TempDir(), "snapshot", "scan.json")
	return cfg
}

// withMissingApps returns a copy of cfg whose app paths do not exist.
func withMissingApps(t *testing.T, cfg *config.Config) *config.Config {
	t.Helper()
	c := *cfg
	c.Apps = append([]config.App{}, cfg.Apps...)
	gone := filepath.Join(t.TempDir(), "gone")
	for i := range c.Apps {
		c.Apps[i].Path = filepath.Join(gone, c.Apps[i].ID)
	}
	return &c
}

func readSnapshotFile(t *testing.T, p string) (*Snapshot, string) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var snap Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		t.Fatal(err)
	}
	return &snap, string(b)
}

func TestBuildWritesSnapshot(t *testing.T) {
	cfg := snapshotConfig(t)
	res, err := Build(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	snap, raw := readSnapshotFile(t, cfg.Snapshot)
	if snap.Version != 1 || snap.BuiltAt.IsZero() || len(snap.Scans) != 3 {
		t.Fatalf("snapshot header: version %d builtAt %v scans %d", snap.Version, snap.BuiltAt, len(snap.Scans))
	}
	if !reflect.DeepEqual(snap.TermLinks, res.TermLinks) {
		t.Fatalf("term links = %v, want %v", snap.TermLinks, res.TermLinks)
	}
	if _, ok := snap.FileSkipped["hive"]; !ok {
		t.Fatalf("fileSkipped has no hive entry: %v", snap.FileSkipped)
	}
	if strings.Contains(raw, "Don't stop") {
		t.Fatal("snapshot contains app source code")
	}
	if len(res.FromSnapshot) != 0 {
		t.Fatalf("live build marked as snapshot: %v", res.FromSnapshot)
	}
}

func TestBuildFallsBackToSnapshot(t *testing.T) {
	cfg := snapshotConfig(t)
	liveOut := t.TempDir()
	live, err := Build(cfg, liveOut)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfg.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapOut := t.TempDir()
	res, err := Build(withMissingApps(t, cfg), snapOut)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"hive", "aladdin", "accounts"}; !reflect.DeepEqual(res.FromSnapshot, want) {
		t.Fatalf("fromSnapshot = %v, want %v", res.FromSnapshot, want)
	}
	if res.SnapshotBuiltAt.IsZero() {
		t.Fatal("snapshotBuiltAt not set")
	}
	for id, sc := range live.Scans {
		got := res.Scans[id]
		if got == nil || len(got.Modules) != len(sc.Modules) || len(got.Routes) != len(sc.Routes) || got.Error != "" {
			t.Fatalf("%s from snapshot differs: %+v", id, got)
		}
	}
	if !reflect.DeepEqual(res.TermLinks, live.TermLinks) {
		t.Fatalf("term links = %v, want %v", res.TermLinks, live.TermLinks)
	}
	assertSameFile(t, liveOut, snapOut, "data/hive.js")
	sync, err := os.ReadFile(filepath.Join(snapOut, "data/sync.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sync), `"mode":"snapshot"`) || !strings.Contains(string(sync), `"snapshotBuiltAt"`) {
		t.Fatalf("sync.js lacks snapshot mode: %s", sync)
	}
	after, err := os.ReadFile(cfg.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("a build from the snapshot must not rewrite the snapshot")
	}
}

func assertSameFile(t *testing.T, a, b, rel string) {
	t.Helper()
	x, err := os.ReadFile(filepath.Join(a, rel))
	if err != nil {
		t.Fatal(err)
	}
	y, err := os.ReadFile(filepath.Join(b, rel))
	if err != nil {
		t.Fatal(err)
	}
	if string(x) != string(y) {
		t.Fatalf("%s differs between live and snapshot builds", rel)
	}
}

func TestBuildMissingAppWithoutSnapshotWarns(t *testing.T) {
	cfg := withMissingApps(t, snapshotConfig(t))
	res, err := Build(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.FromSnapshot) != 0 {
		t.Fatalf("fromSnapshot = %v without a snapshot", res.FromSnapshot)
	}
	found := false
	for _, w := range res.Warnings {
		found = found || strings.HasPrefix(w, "hive: not scanned:")
	}
	if !found {
		t.Fatalf("no 'not scanned' warning: %v", res.Warnings)
	}
	if _, err := os.Stat(cfg.Snapshot); !os.IsNotExist(err) {
		t.Fatalf("snapshot written for a failed scan: %v", err)
	}
}

func TestSyncDataIsLiveWithoutSnapshot(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(snapshotConfig(t), out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "data/sync.js"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"mode"`) {
		t.Fatalf("live sync.js has a mode: %s", b)
	}
}
