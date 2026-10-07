package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/scan"
	"carrybee/handbook/internal/site"
)

func fakeBuild(delay time.Duration, fail bool, page string) BuildFunc {
	return func(cfg *config.Config, out string) (*site.Result, error) {
		time.Sleep(delay)
		if fail {
			return nil, os.ErrPermission
		}
		os.MkdirAll(out, 0o755)
		os.WriteFile(filepath.Join(out, "index.html"), []byte(page), 0o644)
		return &site.Result{Scans: map[string]*scan.AppScan{"hive": {Modules: map[string]*scan.Module{"a": {}}}}}, nil
	}
}

func newTestServer(t *testing.T, b BuildFunc) (*Server, string) {
	dist := filepath.Join(t.TempDir(), "dist")
	os.MkdirAll(dist, 0o755)
	os.WriteFile(filepath.Join(dist, "index.html"), []byte("old"), 0o644)
	return New(&config.Config{}, dist, b), dist
}

func TestStatus(t *testing.T) {
	s, _ := newTestServer(t, fakeBuild(0, false, "new"))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/status", nil))
	var body map[string]any
	json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != 200 || body["mode"] != "local" {
		t.Fatalf("%d %v", rec.Code, body)
	}
}

func TestSyncAtomicSwap(t *testing.T) {
	s, dist := newTestServer(t, fakeBuild(0, false, "new"))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/sync", nil))
	if rec.Code != 200 {
		t.Fatalf("code %d: %s", rec.Code, rec.Body)
	}
	b, _ := os.ReadFile(filepath.Join(dist, "index.html"))
	if string(b) != "new" {
		t.Fatalf("dist not swapped: %q", b)
	}
}

func TestSyncFailureKeepsOldSite(t *testing.T) {
	s, dist := newTestServer(t, fakeBuild(0, true, ""))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/sync", nil))
	if rec.Code != 500 {
		t.Fatalf("code %d", rec.Code)
	}
	b, _ := os.ReadFile(filepath.Join(dist, "index.html"))
	if string(b) != "old" {
		t.Fatalf("old site damaged: %q", b)
	}
}

func TestSyncConflict(t *testing.T) {
	s, _ := newTestServer(t, fakeBuild(300*time.Millisecond, false, "new"))
	h := s.Handler()
	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			time.Sleep(time.Duration(i) * 50 * time.Millisecond)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/sync", nil))
			codes[i] = rec.Code
		}(i)
	}
	wg.Wait()
	if codes[0] != 200 || codes[1] != 409 {
		t.Fatalf("codes = %v, want [200 409]", codes)
	}
}

func TestSyncRejectsGet(t *testing.T) {
	s, _ := newTestServer(t, fakeBuild(0, false, "new"))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/sync", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDiffScans(t *testing.T) {
	mod := func(names ...string) map[string]*scan.Module {
		m := map[string]*scan.Module{}
		for _, n := range names {
			m[n] = &scan.Module{}
		}
		return m
	}
	before := map[string]*scan.AppScan{"hive": {Modules: mod("a", "b"), Routes: map[string]*scan.Route{"x": {}}, APIDomains: map[string][]string{"orders": nil}}}
	after := map[string]*scan.AppScan{"hive": {Modules: mod("b", "c"), Routes: map[string]*scan.Route{"x": {}, "y": {}}, APIDomains: map[string][]string{"orders": nil}}}
	got := DiffScans(before, after)
	want := []Diff{{App: "hive", AddedModules: []string{"c"}, RemovedModules: []string{"a"}, AddedPages: []string{"y"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}
