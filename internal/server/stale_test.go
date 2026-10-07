package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/scan"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.email=test@example.com", "-c", "user.name=test", "-c", "commit.gpgsign=false"}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckStale(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	writeFile(t, filepath.Join(dir, "src", "a.ts"), "export const a = 1;\n")
	writeFile(t, filepath.Join(dir, "src", "node_modules", "x.ts"), "x\n")
	git(t, dir, "add", "src/a.ts")
	git(t, dir, "commit", "-q", "-m", "one")
	builtAt := time.Now().Add(time.Second)
	apps := []config.App{{ID: "hive", Path: dir}}
	last := map[string]*scan.AppScan{"hive": {Commit: currentCommit(dir), ScannedAt: builtAt}}

	if got := CheckStale(apps, last, builtAt); got.Stale || got.Apps[0].NewCommits != 0 || got.Apps[0].ChangedFiles != 0 {
		t.Fatalf("fresh build reported stale: %+v", got)
	}

	writeFile(t, filepath.Join(dir, "README.md"), "readme\n")
	git(t, dir, "add", "README.md")
	git(t, dir, "commit", "-q", "-m", "two")
	got := CheckStale(apps, last, builtAt)
	if !got.Stale || got.Apps[0].NewCommits != 1 || got.Apps[0].CurrentCommit == got.Apps[0].BuiltCommit {
		t.Fatalf("new commit not reported: %+v", got)
	}

	later := builtAt.Add(time.Minute)
	for _, p := range []string{filepath.Join(dir, "src", "a.ts"), filepath.Join(dir, "src", "node_modules", "x.ts")} {
		if err := os.Chtimes(p, later, later); err != nil {
			t.Fatal(err)
		}
	}
	if got := CheckStale(apps, last, builtAt); got.Apps[0].ChangedFiles != 1 {
		t.Fatalf("changed files = %d, want 1 (node_modules skipped)", got.Apps[0].ChangedFiles)
	}
}

func TestCheckStaleWithoutGit(t *testing.T) {
	dir := t.TempDir()
	got := CheckStale([]config.App{{ID: "x", Path: dir}}, nil, time.Now())
	if got.Stale || len(got.Apps) != 1 || got.Apps[0].NewCommits != 0 {
		t.Fatalf("non-git folder must not be stale: %+v", got)
	}
}

func TestStaleCacheExpires(t *testing.T) {
	now := time.Unix(1000, 0)
	s := New(&config.Config{}, t.TempDir(), nil)
	s.now = func() time.Time { return now }
	calls := 0
	s.checkStale = func([]config.App, map[string]*scan.AppScan, time.Time) StaleReport {
		calls++
		return StaleReport{Apps: []AppState{}}
	}
	s.staleness()
	now = now.Add(10 * time.Second)
	s.staleness()
	if calls != 1 {
		t.Fatalf("calls = %d within the cache window, want 1", calls)
	}
	now = now.Add(staleTTL)
	s.staleness()
	if calls != 2 {
		t.Fatalf("calls = %d after the cache window, want 2", calls)
	}
}
