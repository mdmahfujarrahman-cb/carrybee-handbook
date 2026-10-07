package scan

import (
	"fmt"
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
	Skipped      []string          // src-relative paths of unreadable files
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
		RouteFiles: map[string]int{}, Files: map[string]string{}, Skipped: []string{}}
	for m := range modules {
		g.Modules[m] = newModule()
	}
	w := &graphWalker{g: g, src: src, modules: modules, routeOf: routeOf}
	err = filepath.WalkDir(src, w.visit)
	return g, err
}

// graphWalker records one app's imports while walking <app>/src.
type graphWalker struct {
	g       *Graph
	src     string
	modules map[string]bool
	routeOf func([]string) string
}

func isSource(p string) bool { return strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx") }

// visit reads one source file. An unreadable file is listed in Graph.Skipped and the walk goes on.
func (w *graphWalker) visit(p string, d fs.DirEntry, err error) error {
	if err != nil || d.IsDir() || !isSource(p) {
		return err
	}
	relPath, err := filepath.Rel(w.src, p)
	if err != nil {
		return fmt.Errorf("relative path of %s: %w", p, err)
	}
	rel := filepath.ToSlash(relPath)
	raw, err := os.ReadFile(p)
	if err != nil {
		w.g.Skipped = append(w.g.Skipped, rel)
		return nil
	}
	text := StripComments(string(raw))
	w.g.Files[rel] = text
	from := owner(split(relPath), w.modules, w.routeOf)
	if from == "" {
		return nil
	}
	kind, name, _ := strings.Cut(from, ":")
	switch kind {
	case "mod":
		w.g.Modules[name].Files++
	case "route":
		w.g.RouteFiles[name]++
	}
	for _, m := range importRE.FindAllStringSubmatch(text, -1) {
		w.addImport(kind, name, from, resolve(m[1], w.src, p))
	}
	return nil
}

// addImport records one import from the owner kind:name (from) to the src-relative target.
func (w *graphWalker) addImport(kind, name, from, target string) {
	if target == "" {
		return
	}
	to := owner(split(target), w.modules, w.routeOf)
	if to == "" || to == from {
		return
	}
	tk, tn, _ := strings.Cut(to, ":")
	g := w.g
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
