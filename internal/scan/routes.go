package scan

import (
	"strings"

	"carrybee/handbook/internal/config"
)

var layoutGroups = map[string]bool{"(main)": true, "(protected)": true, "(dashboard_layout)": true,
	"(auth)": true, "(public)": true}

// IgnoredRoutePrefixes are page groups that are infrastructure, not product pages.
var IgnoredRoutePrefixes = []string{"api", "debug", "health", "(tests)"}

// RouteNamer returns a function naming the page group for a path under src/app (file name included).
func RouteNamer(app config.App) func(rel []string) string {
	extra := map[string]bool{}
	for _, s := range app.SkipSegments {
		extra[s] = true
	}
	twoLevel := len(app.SkipSegments) > 0
	return func(rel []string) string {
		if len(rel) == 0 {
			return "/"
		}
		dirs := rel[:len(rel)-1]
		var segs []string
		for i, p := range dirs {
			if layoutGroups[p] || strings.HasPrefix(p, "@") || extra[p] {
				continue
			}
			if twoLevel && p == "businesses" && i+1 < len(dirs) && dirs[i+1] == "[businessId]" {
				continue
			}
			if twoLevel && p == "[businessId]" {
				continue
			}
			segs = append(segs, p)
		}
		if len(segs) == 0 {
			return "/"
		}
		if twoLevel {
			var named []string
			for _, s := range segs {
				if !strings.HasPrefix(s, "(") {
					named = append(named, s)
				}
			}
			if len(named) == 0 {
				return segs[0]
			}
			if len(named) > 2 {
				named = named[:2]
			}
			return strings.Join(named, "/")
		}
		if strings.HasPrefix(segs[0], "(") && len(segs) > 1 {
			return segs[0] + "/" + segs[1]
		}
		return segs[0]
	}
}

// pageRoutes drops infrastructure routes.
func pageRoutes(all map[string]*Route) map[string]*Route {
	out := map[string]*Route{}
	for name, r := range all {
		skip := false
		for _, p := range IgnoredRoutePrefixes {
			if name == p || strings.HasPrefix(name, p+"/") {
				skip = true
			}
		}
		if !skip {
			out[name] = r
		}
	}
	return out
}
