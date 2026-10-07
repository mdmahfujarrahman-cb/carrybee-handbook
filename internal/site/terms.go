// Package site renders the handbook into a static directory.
package site

import (
	"regexp"
	"sort"
	"strings"

	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
)

const (
	topPerApp = 3
	maxSpread = 12 // a code name found in more modules than this is too generic to link
	minAlias  = 4
)

// LinkTerms finds the modules whose source mentions each term's code names, then applies pins.
func LinkTerms(terms []content.Term, scans map[string]*scan.AppScan, pins map[string]map[string][]string) []content.Term {
	out := make([]content.Term, len(terms))
	for i, t := range terms {
		t.Where = map[string][]string{}
		for _, app := range t.Apps {
			s := scans[app]
			if s == nil {
				continue
			}
			var found []string
			if s.Files != nil {
				found = rankModules(t.Aka, s)
			}
			if mods := withNamedModules(t.Aka, s, found); len(mods) > 0 {
				t.Where[app] = mods
			}
		}
		for app, mods := range pins[t.Name] {
			t.Where[app] = mods
		}
		out[i] = t
	}
	return out
}

// withNamedModules puts modules whose directory name equals an alias first (Ruling R8),
// then the ranked source matches, keeping at most topPerApp.
func withNamedModules(aka []string, s *scan.AppScan, ranked []string) []string {
	out := make([]string, 0, topPerApp)
	seen := map[string]bool{}
	add := func(m string) {
		if !seen[m] && len(out) < topPerApp {
			seen[m] = true
			out = append(out, m)
		}
	}
	for _, alias := range aka {
		if s.Modules[alias] != nil {
			add(alias)
		}
	}
	for _, m := range ranked {
		add(m)
	}
	return out
}

// countByModule counts regexp matches per module over the app's module sources.
// literal is the alias the regexp wraps; files without it are skipped before the (slow) regexp runs.
func countByModule(re *regexp.Regexp, literal string, s *scan.AppScan) map[string]int {
	found := map[string]int{}
	for rel, text := range s.Files {
		parts := strings.SplitN(rel, "/", 3)
		if len(parts) < 3 || parts[0] != "modules" || s.Modules[parts[1]] == nil || !strings.Contains(text, literal) {
			continue
		}
		if n := len(re.FindAllStringIndex(text, -1)); n > 0 {
			found[parts[1]] += n
		}
	}
	return found
}

func rankModules(aka []string, s *scan.AppScan) []string {
	hits := map[string]int{}
	for _, alias := range aka {
		if len(alias) < minAlias {
			continue
		}
		re := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(alias) + `($|[^A-Za-z0-9_])`)
		found := countByModule(re, alias, s)
		if len(found) == 0 || len(found) > maxSpread {
			continue
		}
		for m, n := range found {
			hits[m] += n
		}
	}
	mods := make([]string, 0, len(hits))
	for m := range hits {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool {
		if hits[mods[i]] != hits[mods[j]] {
			return hits[mods[i]] > hits[mods[j]]
		}
		return mods[i] < mods[j]
	})
	if len(mods) > topPerApp {
		mods = mods[:topPerApp]
	}
	return mods
}
