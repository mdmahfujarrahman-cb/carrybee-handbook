// Package server serves the built handbook and rebuilds it on demand.
package server

import (
	"sort"

	"carrybee/handbook/internal/scan"
)

type Diff struct {
	App            string   `json:"app"`
	AddedModules   []string `json:"addedModules,omitempty"`
	RemovedModules []string `json:"removedModules,omitempty"`
	AddedPages     []string `json:"addedPages,omitempty"`
	RemovedPages   []string `json:"removedPages,omitempty"`
	AddedAPIs      []string `json:"addedApis,omitempty"`
	RemovedAPIs    []string `json:"removedApis,omitempty"`
}

func keys[V any](m map[string]V) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func added(before, after map[string]bool) []string {
	var out []string
	for k := range after {
		if !before[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func DiffScans(before, after map[string]*scan.AppScan) []Diff {
	apps := make([]string, 0, len(after))
	for a := range after {
		apps = append(apps, a)
	}
	sort.Strings(apps)
	out := []Diff{}
	for _, a := range apps {
		b, n := before[a], after[a]
		if b == nil {
			b = &scan.AppScan{}
		}
		d := Diff{App: a,
			AddedModules: added(keys(b.Modules), keys(n.Modules)), RemovedModules: added(keys(n.Modules), keys(b.Modules)),
			AddedPages: added(keys(b.Routes), keys(n.Routes)), RemovedPages: added(keys(n.Routes), keys(b.Routes)),
			AddedAPIs: added(keys(b.APIDomains), keys(n.APIDomains)), RemovedAPIs: added(keys(n.APIDomains), keys(b.APIDomains))}
		if len(d.AddedModules)+len(d.RemovedModules)+len(d.AddedPages)+len(d.RemovedPages)+len(d.AddedAPIs)+len(d.RemovedAPIs) > 0 {
			out = append(out, d)
		}
	}
	return out
}

// DiffTermLinks lists "term → app/module" links present after but not before, sorted.
// It returns an empty slice (never nil), including on the first sync when before is nil.
func DiffTermLinks(before, after map[string]map[string][]string) []string {
	out := []string{}
	if before == nil {
		return out
	}
	for term, apps := range after {
		for app, mods := range apps {
			old := map[string]bool{}
			for _, m := range before[term][app] {
				old[m] = true
			}
			for _, m := range mods {
				if !old[m] {
					out = append(out, term+" → "+app+"/"+m)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}
