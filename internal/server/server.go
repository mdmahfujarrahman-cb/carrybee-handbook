package server

import (
	"encoding/json"
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
	builtAt time.Time
}

func New(cfg *config.Config, dist string, build BuildFunc) *Server {
	return &Server{cfg: cfg, dist: dist, build: build}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
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
	writeJSON(w, 200, map[string]any{"mode": "local", "builtAt": s.builtAt, "commits": commits, "syncing": s.syncing})
}

func (s *Server) sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "use POST"})
		return
	}
	s.mu.Lock()
	if s.syncing {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "sync already running"})
		return
	}
	s.syncing = true
	before := s.last
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.syncing = false; s.mu.Unlock() }()

	start := time.Now()
	res, err := s.rebuild()
	if err != nil {
		writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.mu.Lock()
	s.last, s.builtAt = res.Scans, time.Now().UTC()
	s.mu.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true, "diff": DiffScans(before, res.Scans),
		"missing": res.Missing, "warnings": res.Warnings, "seconds": time.Since(start).Seconds()})
}

// rebuild builds into a sibling temp dir and swaps it into place, keeping the old site on failure.
func (s *Server) rebuild() (*site.Result, error) {
	parent := filepath.Dir(s.dist)
	tmp, err := os.MkdirTemp(parent, ".dist-new-")
	if err != nil {
		return nil, err
	}
	res, err := s.build(s.cfg, tmp)
	if err != nil {
		os.RemoveAll(tmp)
		return nil, fmt.Errorf("build failed, site unchanged: %w", err)
	}
	old := s.dist + ".old"
	os.RemoveAll(old)
	if err := os.Rename(s.dist, old); err != nil && !os.IsNotExist(err) {
		os.RemoveAll(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, s.dist); err != nil {
		os.Rename(old, s.dist)
		return nil, err
	}
	os.RemoveAll(old)
	return res, nil
}

// ListenAndServe builds once, then serves on 127.0.0.1:port.
func (s *Server) ListenAndServe(port int) error {
	res, err := s.rebuild()
	if err != nil {
		return err
	}
	s.last, s.builtAt = res.Scans, time.Now().UTC()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("Carrybee Handbook on http://%s (Sync enabled)", addr)
	return http.ListenAndServe(addr, s.Handler())
}
