package server

import (
	"io/fs"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/scan"
)

// staleTTL is how long a staleness report is reused before git and the file tree are checked again.
const staleTTL = 30 * time.Second

// AppState says how far one app's code has moved since the handbook was built.
type AppState struct {
	ID            string `json:"id"`
	BuiltCommit   string `json:"builtCommit"`
	CurrentCommit string `json:"currentCommit"`
	NewCommits    int    `json:"newCommits"`
	ChangedFiles  int    `json:"changedFiles"`
}

// StaleReport is the staleness of every configured app. Apps is never nil.
type StaleReport struct {
	Stale bool       `json:"stale"`
	Apps  []AppState `json:"apps"`
}

// skippedDirs are not app source, so edits there never make the handbook stale.
var skippedDirs = map[string]bool{"node_modules": true, ".next": true}

func gitOutput(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

// currentCommit returns the short HEAD commit, or "" when dir is not a git checkout.
func currentCommit(dir string) string {
	out, err := gitOutput(dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		log.Printf("stale check: commit of %s: %v", dir, err)
		return ""
	}
	return out
}

// newCommits counts commits after built on HEAD; 0 when built is unknown or git fails.
func newCommits(dir, built string) int {
	if built == "" {
		return 0
	}
	out, err := gitOutput(dir, "rev-list", "--count", built+"..HEAD")
	if err != nil {
		log.Printf("stale check: commits since %s in %s: %v", built, dir, err)
		return 0
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		log.Printf("stale check: commit count %q in %s: %v", out, dir, err)
		return 0
	}
	return n
}

// changedFiles counts .ts/.tsx files under <dir>/src modified after since.
func changedFiles(dir string, since time.Time) int {
	n := 0
	err := filepath.WalkDir(filepath.Join(dir, "src"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".ts") && !strings.HasSuffix(p, ".tsx") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(since) {
			n++
		}
		return nil
	})
	if err != nil {
		log.Printf("stale check: walk %s: %v", dir, err)
	}
	return n
}

// CheckStale compares each app's checkout with the last build: last holds the scans the site was
// built from (built commit and scan time per app) and builtAt is used when an app has no scan time.
func CheckStale(apps []config.App, last map[string]*scan.AppScan, builtAt time.Time) StaleReport {
	r := StaleReport{Apps: make([]AppState, 0, len(apps))}
	for _, a := range apps {
		st := AppState{ID: a.ID}
		since := builtAt
		if sc := last[a.ID]; sc != nil {
			st.BuiltCommit = sc.Commit
			if !sc.ScannedAt.IsZero() {
				since = sc.ScannedAt
			}
		}
		st.CurrentCommit = currentCommit(a.Path)
		st.NewCommits = newCommits(a.Path, st.BuiltCommit)
		st.ChangedFiles = changedFiles(a.Path, since)
		if st.CurrentCommit != st.BuiltCommit || st.NewCommits > 0 || st.ChangedFiles > 0 {
			r.Stale = true
		}
		r.Apps = append(r.Apps, st)
	}
	return r
}

// staleness returns the cached report, recomputing it once it is older than staleTTL.
// The git and file checks run outside the lock so a slow check never blocks Sync.
func (s *Server) staleness() StaleReport {
	now := s.now()
	s.mu.Lock()
	if s.stale != nil && now.Sub(s.staleAt) < staleTTL {
		r := *s.stale
		s.mu.Unlock()
		return r
	}
	last, builtAt := s.last, s.builtAt
	s.mu.Unlock()
	r := s.checkStale(s.cfg.Apps, last, builtAt)
	s.mu.Lock()
	s.stale, s.staleAt = &r, now
	s.mu.Unlock()
	return r
}
