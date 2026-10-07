package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	mu      sync.Mutex // guards syncing, last, builtAt
	syncing bool
	last    map[string]*scan.AppScan
	links   map[string]map[string][]string
	builtAt time.Time
}

func New(cfg *config.Config, dist string, build BuildFunc) *Server {
	return &Server{cfg: cfg, dist: dist, build: build}
}

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
	s.mu.Lock()
	defer s.mu.Unlock()
	commits := map[string]string{}
	for id, sc := range s.last {
		commits[id] = sc.Commit
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": "local", "builtAt": s.builtAt, "commits": commits, "syncing": s.syncing})
}

func (s *Server) sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{keyOK: false, keyError: "use POST"})
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
	s.mu.Lock()
	s.last, s.links, s.builtAt = res.Scans, res.TermLinks, time.Now().UTC()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{keyOK: true, "diff": DiffScans(before, res.Scans),
		"terms":   DiffTermLinks(beforeLinks, res.TermLinks),
		"missing": res.Missing, "warnings": res.Warnings, "seconds": time.Since(start).Seconds()})
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

// ListenAndServe builds once, then serves on 127.0.0.1:port.
func (s *Server) ListenAndServe(port int) error {
	res, err := s.rebuild()
	if err != nil {
		return err
	}
	s.last, s.links, s.builtAt = res.Scans, res.TermLinks, time.Now().UTC()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("Carrybee Handbook on http://%s (Sync enabled)", addr)
	return http.ListenAndServe(addr, s.Handler())
}
