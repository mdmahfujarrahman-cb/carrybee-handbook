package site

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
)

// snapshotVersion is the format version of snapshot/scan.json.
const snapshotVersion = 1

// Snapshot is the structure found by the last complete live scan. It lets the handbook be built
// where the app repos are missing. It never holds source code: AppScan.Files is not serialised.
type Snapshot struct {
	Version     int                            `json:"version"`
	BuiltAt     time.Time                      `json:"builtAt"`
	Scans       map[string]*scan.AppScan       `json:"scans"`
	TermLinks   map[string]map[string][]string `json:"termLinks"`
	FileSkipped map[string][]string            `json:"fileSkipped"`
}

// LoadSnapshot reads the snapshot at p. A missing file (or an empty p) is not an error: it returns nil.
func LoadSnapshot(p string) (*Snapshot, error) {
	if p == "" {
		return nil, nil
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot %s: %w", p, err)
	}
	if snap.Version != snapshotVersion {
		return nil, fmt.Errorf("snapshot %s has version %d, want %d", p, snap.Version, snapshotVersion)
	}
	return &snap, nil
}

// AppPresent reports whether an app's code folder exists.
func AppPresent(a config.App) bool {
	info, err := os.Stat(a.Path)
	return err == nil && info.IsDir()
}

// loadSnapshotIfNeeded reads the snapshot only when some app's code folder is missing.
func loadSnapshotIfNeeded(cfg *config.Config) (*Snapshot, error) {
	for _, a := range cfg.Apps {
		if !AppPresent(a) {
			return LoadSnapshot(cfg.Snapshot)
		}
	}
	return nil, nil
}

// scanApps scans the apps whose code is present and takes the others from the snapshot when it has
// them. It returns the scans, the ids taken from the snapshot (in config order) and the snapshot.
func scanApps(cfg *config.Config) (map[string]*scan.AppScan, []string, *Snapshot, error) {
	snap, err := loadSnapshotIfNeeded(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	live := *cfg
	live.Apps = nil
	var fromSnap []string
	for _, a := range cfg.Apps {
		if snap != nil && snap.Scans[a.ID] != nil && !AppPresent(a) {
			fromSnap = append(fromSnap, a.ID)
			continue
		}
		live.Apps = append(live.Apps, a)
	}
	scans := scan.ScanAll(&live)
	for _, id := range fromSnap {
		scans[id] = snap.Scans[id]
	}
	return scans, fromSnap, snap, nil
}

// applySnapshotLinks replaces the term links of snapshot apps (whose source cannot be searched)
// with the links stored in the snapshot; current pins still win.
func applySnapshotLinks(terms []content.Term, snap *Snapshot, fromSnap []string, pins map[string]map[string][]string) {
	for i := range terms {
		t := &terms[i]
		for _, app := range fromSnap {
			delete(t.Where, app)
			if mods := snap.TermLinks[t.ID][app]; len(mods) > 0 {
				t.Where[app] = mods
			}
			if pinned, ok := pins[t.Name][app]; ok {
				t.Where[app] = pinned
			}
		}
	}
}

// allScannedLive reports whether every configured app was scanned from its code without an error.
func (s *site) allScannedLive() bool {
	if len(s.res.FromSnapshot) > 0 {
		return false
	}
	for _, a := range s.cfg.Apps {
		if sc := s.scans[a.ID]; sc == nil || sc.Error != "" {
			return false
		}
	}
	return true
}

// saveSnapshot stores the scan after a complete live build, so copies without the repos can rebuild.
func (s *site) saveSnapshot() error {
	if s.cfg.Snapshot == "" || !s.allScannedLive() {
		return nil
	}
	skipped := make(map[string][]string, len(s.scans))
	for id, sc := range s.scans {
		skipped[id] = orEmpty(sc.Skipped)
	}
	snap := Snapshot{Version: snapshotVersion, BuiltAt: time.Now().UTC(), Scans: s.scans,
		TermLinks: s.res.TermLinks, FileSkipped: skipped}
	b, err := json.MarshalIndent(snap, "", " ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.cfg.Snapshot, append(b, '\n'))
}

// orEmpty keeps empty lists as [] in JSON instead of null.
func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// writeFileAtomic writes b to a temp file next to p and renames it over p.
func writeFileAtomic(p string, b []byte) (err error) {
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".scan-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		if err == nil {
			return
		}
		if rmErr := os.Remove(tmp); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			err = errors.Join(err, rmErr)
		}
	}()
	_, err = f.Write(b)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Chmod(tmp, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
