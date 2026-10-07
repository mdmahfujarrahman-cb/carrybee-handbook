package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/scan"
	"carrybee/handbook/internal/site"
)

type BuildFunc func(cfg *config.Config, out string) (*site.Result, error)

type Server struct {
	cfg     *config.Config
	dist    string
	build   BuildFunc
	mu      sync.Mutex // guards syncing, last, links, builtAt, stale, staleAt
	syncing bool
	last    map[string]*scan.AppScan
	links   map[string]map[string][]string
	builtAt time.Time
	stale   *StaleReport // cached staleness, nil after a build
	staleAt time.Time

	now        func() time.Time
	checkStale func([]config.App, map[string]*scan.AppScan, time.Time) StaleReport
}

func New(cfg *config.Config, dist string, build BuildFunc) *Server {
	return &Server{cfg: cfg, dist: dist, build: build, now: time.Now, checkStale: CheckStale}
}

// readHeaderTimeout bounds slow clients (gosec G112).
const readHeaderTimeout = 5 * time.Second

const (
	keyOK    = "ok"
	keyError = "error"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", s.status)
	mux.HandleFunc("/api/sync", s.sync)
	mux.Handle("/", http.FileServer(http.Dir(s.dist)))
	return mux
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	can, reason := s.canSync()
	stale := StaleReport{Apps: []AppState{}}
	if can {
		stale = s.staleness()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	commits := map[string]string{}
	for id, sc := range s.last {
		commits[id] = sc.Commit
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": "local", "builtAt": s.builtAt, "commits": commits,
		"syncing": s.syncing, "stale": stale.Stale, "apps": stale.Apps, "canSync": can, "reason": reason})
}

// crossOrigin reports a browser request sent from another site (Origin set and not this host).
func crossOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	return err != nil || u.Host != r.Host
}

func (s *Server) sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{keyOK: false, keyError: "use POST"})
		return
	}
	if crossOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{keyOK: false, keyError: "sync is only allowed from this site"})
		return
	}
	if can, reason := s.canSync(); !can {
		writeJSON(w, http.StatusConflict, map[string]any{keyOK: false, keyError: reason})
		return
	}
	s.mu.Lock()
	if s.syncing {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]any{keyOK: false, keyError: "sync already running"})
		return
	}
	s.syncing = true
	before, beforeLinks := s.last, s.links
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.syncing = false; s.mu.Unlock() }()

	start := time.Now()
	res, err := s.rebuild()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{keyOK: false, keyError: err.Error()})
		return
	}
	s.setBuilt(res)
	writeJSON(w, http.StatusOK, map[string]any{keyOK: true, "diff": DiffScans(before, res.Scans),
		"terms":   DiffTermLinks(beforeLinks, res.TermLinks),
		"missing": orEmpty(res.Missing), "warnings": orEmpty(res.Warnings), "seconds": time.Since(start).Seconds()})
}

// orEmpty keeps empty lists as [] in JSON instead of null.
func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// rebuild builds into a sibling temp dir and swaps it into place, keeping the old site on failure.
func (s *Server) rebuild() (res *site.Result, err error) {
	tmp, err := os.MkdirTemp(filepath.Dir(s.dist), ".dist-new-")
	if err != nil {
		return nil, err
	}
	swapped := false
	defer func() {
		if !swapped {
			if rmErr := os.RemoveAll(tmp); rmErr != nil {
				log.Printf("cleanup %s: %v", tmp, rmErr)
			}
		}
	}()
	res, err = s.build(s.cfg, tmp)
	if err != nil {
		return nil, fmt.Errorf("build failed, site unchanged: %w", err)
	}
	if err := s.swap(tmp); err != nil {
		return nil, err
	}
	swapped = true
	return res, nil
}

// swap replaces dist with tmp using two renames, rolling back if the second fails.
func (s *Server) swap(tmp string) error {
	old := s.dist + ".old"
	if err := os.RemoveAll(old); err != nil {
		return fmt.Errorf("clear previous backup, site unchanged: %w", err)
	}
	if err := os.Rename(s.dist, old); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("move current site aside, site unchanged: %w", err)
	}
	if err := os.Rename(tmp, s.dist); err != nil {
		if rbErr := os.Rename(old, s.dist); rbErr != nil && !os.IsNotExist(rbErr) {
			return errors.Join(err, fmt.Errorf("rollback failed, site may be missing (previous copy in %s): %w", old, rbErr))
		}
		return fmt.Errorf("install new site, site unchanged: %w", err)
	}
	if err := os.RemoveAll(old); err != nil {
		log.Printf("cleanup %s: %v", old, err)
	}
	return nil
}

// ListenAndServe builds once (from the snapshot for apps whose code is missing), then serves on 127.0.0.1:port.
func (s *Server) ListenAndServe(port int) error {
	res, err := s.rebuild()
	if err != nil {
		return err
	}
	s.setBuilt(res)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if can, reason := s.canSync(); can {
		log.Printf("Carrybee Handbook on http://%s (Sync enabled)", addr)
	} else {
		log.Printf("Carrybee Handbook on http://%s (Sync disabled: %s)", addr, reason)
	}
	return s.httpServer(addr).ListenAndServe()
}

func (s *Server) httpServer(addr string) *http.Server {
	return &http.Server{Addr: addr, Handler: s.Handler(), ReadHeaderTimeout: readHeaderTimeout}
}

// setBuilt records a finished build and drops the cached staleness report.
func (s *Server) setBuilt(res *site.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last, s.links, s.builtAt = res.Scans, res.TermLinks, s.now().UTC()
	s.stale = nil
}
