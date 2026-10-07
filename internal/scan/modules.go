package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"carrybee/handbook/internal/config"
)

type Module struct {
	Files         int            `json:"files"`
	Deps          map[string]int `json:"deps"`
	UsedByModules map[string]int `json:"usedByMods"`
	UsedByRoutes  map[string]int `json:"usedByRoutes"`
	APIs          map[string]int `json:"apis"`
	Shared        map[string]int `json:"shared"`
}

func newModule() *Module {
	return &Module{Deps: map[string]int{}, UsedByModules: map[string]int{}, UsedByRoutes: map[string]int{},
		APIs: map[string]int{}, Shared: map[string]int{}}
}

type Graph struct {
	Modules      map[string]*Module
	RouteImports map[string]map[string]int // route -> module -> count
	RouteFiles   map[string]int
	Files        map[string]string // rel path under src -> comment-stripped source
}

var importRE = regexp.MustCompile(`(?:from|import)\s*\(?\s*['"]([^'"]+)['"]`)

var sharedDirs = map[string]bool{"components": true, "lib": true, "hooks": true, "contexts": true,
	"services": true, "types": true, "validations": true}

// owner names what a src-relative path belongs to: "mod:<name>", "route:<name>", "shared:<dir>" or "".
func owner(rel []string, modules map[string]bool, routeOf func([]string) string) string {
	if len(rel) == 0 {
		return ""
	}
	switch {
	case rel[0] == "modules" && len(rel) > 1 && modules[rel[1]]:
		return "mod:" + rel[1]
	case rel[0] == "app":
		return "route:" + routeOf(rel[1:])
	case sharedDirs[rel[0]]:
		if len(rel) > 1 {
			return "shared:" + rel[0] + "/" + rel[1]
		}
		return "shared:" + rel[0]
	}
	return ""
}

func split(p string) []string { return strings.Split(filepath.ToSlash(p), "/") }

// BuildGraph walks <app>/src and records module-to-module and route-to-module imports.
func BuildGraph(app config.App, routeOf func(rel []string) string) (*Graph, error) {
	src := filepath.Join(app.Path, "src")
	modules := map[string]bool{}
	entries, err := os.ReadDir(filepath.Join(src, "modules"))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			modules[e.Name()] = true
		}
	}
	g := &Graph{Modules: map[string]*Module{}, RouteImports: map[string]map[string]int{},
		RouteFiles: map[string]int{}, Files: map[string]string{}}
	for m := range modules {
		g.Modules[m] = newModule()
	}
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !(strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx")) {
			return err
		}
		relPath, _ := filepath.Rel(src, p)
		raw, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil // unreadable file: skip, counted by caller via Files
		}
		text := StripComments(string(raw))
		g.Files[filepath.ToSlash(relPath)] = text
		from := owner(split(relPath), modules, routeOf)
		if from == "" {
			return nil
		}
		kind, name, _ := strings.Cut(from, ":")
		switch kind {
		case "mod":
			g.Modules[name].Files++
		case "route":
			g.RouteFiles[name]++
		}
		for _, m := range importRE.FindAllStringSubmatch(text, -1) {
			target := resolve(m[1], src, p)
			if target == "" {
				continue
			}
			to := owner(split(target), modules, routeOf)
			if to == "" || to == from {
				continue
			}
			tk, tn, _ := strings.Cut(to, ":")
			switch {
			case kind == "mod" && tk == "mod":
				g.Modules[name].Deps[tn]++
				g.Modules[tn].UsedByModules[name]++
			case kind == "mod" && tk == "shared":
				g.Modules[name].Shared[tn]++
			case kind == "route" && tk == "mod":
				if g.RouteImports[name] == nil {
					g.RouteImports[name] = map[string]int{}
				}
				g.RouteImports[name][tn]++
				g.Modules[tn].UsedByRoutes[name]++
			}
		}
		return nil
	})
	return g, err
}

// resolve turns an import specifier into a path relative to src, or "" for packages.
func resolve(spec, src, file string) string {
	var abs string
	switch {
	case strings.HasPrefix(spec, "@/"):
		abs = filepath.Join(src, spec[2:])
	case strings.HasPrefix(spec, "."):
		abs = filepath.Join(filepath.Dir(file), spec)
	default:
		return ""
	}
	rel, err := filepath.Rel(src, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return rel
}
