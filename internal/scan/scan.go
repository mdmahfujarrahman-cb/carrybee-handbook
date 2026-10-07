package scan

import (
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"carrybee/handbook/internal/config"
)

type Route struct {
	Files int            `json:"files"`
	Mods  map[string]int `json:"mods"`
}

type AppScan struct {
	App        string              `json:"app"`
	Label      string              `json:"label"`
	Audience   string              `json:"audience"`
	Commit     string              `json:"commit"`
	ScannedAt  time.Time           `json:"scannedAt"`
	Error      string              `json:"error,omitempty"`
	Modules    map[string]*Module  `json:"modules"`
	Routes     map[string]*Route   `json:"routes"`
	APIDomains map[string][]string `json:"apiDomains"`
	Statuses   []Status            `json:"statuses"`
	// TextStatuses are the string order statuses (orderStatusSchema and friends), never null.
	TextStatuses []string `json:"textStatuses"`
	// Skipped lists src-relative files that could not be read.
	Skipped []string          `json:"skipped,omitempty"`
	Cycles  [][]string        `json:"cycles"`
	Files   map[string]string `json:"-"`
}

func gitCommit(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ScanApp scans one app. Failures are reported in AppScan.Error, never as a nil result.
func ScanApp(app config.App) *AppScan {
	s := &AppScan{App: app.ID, Label: app.Label, Audience: app.Audience, ScannedAt: time.Now().UTC(),
		Modules: map[string]*Module{}, Routes: map[string]*Route{}, APIDomains: map[string][]string{},
		Statuses: []Status{}, TextStatuses: []string{}, Cycles: [][]string{}}
	g, err := BuildGraph(app, RouteNamer(app))
	if err != nil {
		s.Error = err.Error()
		return s
	}
	s.Commit = gitCommit(app.Path)
	s.Files = g.Files
	s.Skipped = g.Skipped
	s.TextStatuses = textStatuses(g.Files)
	s.Modules = g.Modules
	for name, files := range g.RouteFiles {
		s.Routes[name] = &Route{Files: files, Mods: g.RouteImports[name]}
		if s.Routes[name].Mods == nil {
			s.Routes[name].Mods = map[string]int{}
		}
	}
	s.Routes = pageRoutes(s.Routes)
	// A page group must import at least one module (matches the prototype's counting).
	for name, r := range s.Routes {
		if len(r.Mods) == 0 {
			delete(s.Routes, name)
		}
	}
	for _, m := range s.Modules {
		for r := range m.UsedByRoutes {
			if s.Routes[r] == nil {
				delete(m.UsedByRoutes, r)
			}
		}
	}
	domainMods := map[string]map[string]bool{}
	deps := map[string]map[string]int{}
	for rel, text := range g.Files {
		parts := split(rel)
		if len(parts) > 1 && parts[0] == "modules" && s.Modules[parts[1]] != nil {
			for d, n := range apiDomains(text, app.NestedAPI) {
				s.Modules[parts[1]].APIs[d] += n
				if domainMods[d] == nil {
					domainMods[d] = map[string]bool{}
				}
				domainMods[d][parts[1]] = true
			}
		}
		if st := parseStatuses(text); len(st) > len(s.Statuses) {
			s.Statuses = st
		}
	}
	for d, mods := range domainMods {
		for m := range mods {
			s.APIDomains[d] = append(s.APIDomains[d], m)
		}
		sort.Strings(s.APIDomains[d])
	}
	for name, m := range s.Modules {
		deps[name] = m.Deps
	}
	s.Cycles = Cycles(deps)
	return s
}

// ScanAll scans every configured app concurrently.
func ScanAll(cfg *config.Config) map[string]*AppScan {
	out := make(map[string]*AppScan, len(cfg.Apps))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, app := range cfg.Apps {
		wg.Add(1)
		go func(a config.App) {
			defer wg.Done()
			s := ScanApp(a)
			mu.Lock()
			out[a.ID] = s
			mu.Unlock()
		}(app)
	}
	wg.Wait()
	return out
}
