# Carrybee Handbook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `carrybee-handbook`, a Go tool that scans the hive, aladdin and accounts frontends and generates a branded, searchable onboarding handbook (static `dist/`) with a local Sync server.

**Architecture:** One Go binary with three commands. `internal/scan` reads the three Next.js repos into JSON-able structs; `internal/content` loads hand-written Markdown and curated JSON; `internal/site` renders everything through embedded `html/template` files into `dist/`; `internal/server` serves `dist/` and exposes `POST /api/sync` to rescan and rebuild. All page data the browser needs is emitted as `.js` files (`window.HB = …`) so the site works when opened from disk (`file://`) as well as when hosted.

**Tech Stack:** Go 1.24 (standard library), `github.com/yuin/goldmark` v1.7.8 for Markdown, vanilla JS + CSS (no Node), d3 v7.9.0 from cdnjs for the network view only.

**Spec:** `docs/specs/2026-10-07-carrybee-handbook-design.md`

## Global Constraints

- Go module path: `carrybee/handbook`; Go version floor `go 1.24`.
- Only external Go dependency: `github.com/yuin/goldmark v1.7.8`. Concurrency uses `sync.WaitGroup`, not errgroup.
- Commands: `handbook build`, `handbook serve [--port 4321]`, `handbook check`. Server binds `127.0.0.1` only.
- Config file `handbook.json`; app paths relative to the config file; `--apps-root` flag overrides the parent directory of all app paths.
- Brand primary `oklch(0.865 0.177 90.382)` with text `oklch(0.2 0 89.876)` on it; yellow is only ever behind dark text.
- Light and dark themes: tokens on `:root`, redefined under `@media (prefers-color-scheme: dark)` guarded by `:root:not([data-theme="light"])` and again under `:root[data-theme="dark"]`.
- Content column about 72ch; works at 400px width; no horizontal page scroll; visible focus; `prefers-reduced-motion` respected.
- Sync never writes into `content/` or `data/`; it only regenerates `dist/`.
- Every internal link is relative (site may be served from a sub-path such as `/carrybee-handbook/`).
- Parity numbers from the prototype scan: hive 76 modules / 82 page groups / 4 cycles; aladdin 18 / 54 / 1; accounts 10 / 10 / 0.
- Commit messages end with: `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- Prototype sources to port live in `../codebase-maps-backup-20261007-1303/_generator/` (abbreviated **BK** below).

## Review Focus

- **Opened from disk (`file://`)**: search, term popovers and app maps must still work → all browser data ships as `<script src="data/*.js">`, never `fetch()`; pinned by Task 6 test `TestNoFetchInTheme` and Task 11 headless check.
- **One app path missing or not a git repo**: build still produces the other apps; the missing app shows "not scanned"; `check` exits non-zero → Task 3 test `TestScanAllMissingApp`.
- **Sync clicked twice / while a build runs**: second request gets 409 and `dist/` is never half-written → Task 9 tests `TestSyncConflict` and `TestSyncAtomicSwap`.
- **Content references an unknown term/module/status**: `build` warns, `check` fails with file and reference named → Task 5 test `TestCheckUnknownReference`.
- **Hosted under a sub-path**: every generated internal link is relative to the page → Task 6 test `TestLinksAreRelative`.

---

## File map

```
carrybee-handbook/
  go.mod                               module + goldmark
  handbook.json                        config (Task 1)
  cmd/handbook/main.go                 CLI (Task 1, extended in 6 and 9)
  internal/config/config.go            Load config (Task 1)
  internal/scan/source.go              strip comments, read files (Task 2)
  internal/scan/modules.go             module discovery + import graph (Task 2)
  internal/scan/routes.go              page-group naming (Task 3)
  internal/scan/api.go                 API_ROUTES domains (Task 3)
  internal/scan/statuses.go            TransferStatus enum (Task 3)
  internal/scan/cycles.go              Tarjan SCC (Task 3)
  internal/scan/scan.go                AppScan, ScanApp, ScanAll (Task 3)
  internal/site/terms.go               LinkTerms (Task 4)
  internal/scan/testdata/app/...       fixture Next.js tree (Tasks 2-4)
  internal/content/data.go             curated JSON types + loader (Task 4)
  internal/content/pages.go            Markdown pages + front matter (Task 5)
  internal/content/shortcodes.go       {{term}} {{module}} {{status}} + callouts (Task 5)
  internal/content/check.go            reference validation (Task 5)
  internal/site/build.go               orchestrates a full build into a dir (Task 6)
  internal/site/render.go              templates, nav, relative links (Task 6)
  internal/site/search.go              search index (Task 6)
  internal/site/appmap.go              per-app data files + map pages (Task 7)
  internal/site/generated.go           API index, statuses, terms pages (Task 8)
  internal/site/linkcheck.go           dist link checker (Task 6)
  internal/server/server.go            serve + /api/status + /api/sync (Task 9)
  internal/server/diff.go              scan diff (Task 9)
  theme/embed.go                       go:embed of theme files (Task 6)
  theme/templates/*.html               layout + page templates (Task 6, 7, 8)
  theme/assets/handbook.css            tokens + components (Task 6)
  theme/assets/handbook.js             search, theme, popovers, sync pill (Task 6, 9)
  theme/assets/explorer.js             app map explorer (Task 7)
  data/*.json                          curated data migrated from BK (Task 4)
  content/**/*.md                      hand-written pages (Task 10)
  .gitlab-ci.yml  Dockerfile  README.md  (Task 11)
```

---

### Task 1: Module skeleton, config and CLI

**Files:**
- Create: `go.mod`, `handbook.json`, `internal/config/config.go`, `internal/config/config_test.go`, `cmd/handbook/main.go`

**Interfaces:**
- Produces: `config.Load(path string, appsRoot string) (*config.Config, error)`;
  `type Config struct { SiteTitle string; BaseURL string; Apps []App; Dir string }`;
  `type App struct { ID, Path, Label, Audience string; NestedAPI []string; SkipSegments []string }` (Path is absolute after Load).

- [ ] **Step 1: Create go.mod and handbook.json**

```bash
cd /Users/tvs/Workspace/new-work/carrybee-handbook
go mod init carrybee/handbook
go get github.com/yuin/goldmark@v1.7.8
```

`handbook.json`:
```json
{
  "siteTitle": "Carrybee Handbook",
  "baseURL": "/",
  "apps": [
    {"id": "hive", "path": "../hive", "label": "hive", "audience": "ops"},
    {"id": "aladdin", "path": "../aladdin", "label": "aladdin", "audience": "merchant",
     "nestedAPI": ["business"], "skipSegments": ["(business)", "(merchant)"]},
    {"id": "accounts", "path": "../accounts", "label": "accounts", "audience": "finance"}
  ]
}
```

- [ ] **Step 2: Write the failing test** `internal/config/config_test.go`

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "handbook.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadResolvesPathsRelativeToConfig(t *testing.T) {
	p := writeConfig(t, `{"siteTitle":"X","apps":[{"id":"hive","path":"../hive"}]}`)
	c, err := Load(p, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(filepath.Join(filepath.Dir(p), "../hive"))
	if c.Apps[0].Path != want {
		t.Fatalf("path = %q, want %q", c.Apps[0].Path, want)
	}
	if c.Apps[0].Label != "hive" {
		t.Fatalf("label should default to id, got %q", c.Apps[0].Label)
	}
}

func TestLoadAppsRootOverride(t *testing.T) {
	p := writeConfig(t, `{"apps":[{"id":"hive","path":"../hive"}]}`)
	c, err := Load(p, "/ci/repos")
	if err != nil {
		t.Fatal(err)
	}
	if c.Apps[0].Path != "/ci/repos/hive" {
		t.Fatalf("got %q", c.Apps[0].Path)
	}
}

func TestLoadRejectsDuplicateIDs(t *testing.T) {
	p := writeConfig(t, `{"apps":[{"id":"a","path":"x"},{"id":"a","path":"y"}]}`)
	if _, err := Load(p, ""); err == nil {
		t.Fatal("expected duplicate id error")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/config/`
Expected: FAIL — `undefined: Load`

- [ ] **Step 4: Implement** `internal/config/config.go`

```go
// Package config loads handbook.json.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type App struct {
	ID           string   `json:"id"`
	Path         string   `json:"path"`
	Label        string   `json:"label"`
	Audience     string   `json:"audience"`
	NestedAPI    []string `json:"nestedAPI"`
	SkipSegments []string `json:"skipSegments"`
}

type Config struct {
	SiteTitle string `json:"siteTitle"`
	BaseURL   string `json:"baseURL"`
	Apps      []App  `json:"apps"`
	Dir       string `json:"-"` // directory holding handbook.json
}

// Load reads the config. appsRoot, when set, replaces the parent directory of every app path.
func Load(path, appsRoot string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	c.Dir = filepath.Dir(abs)
	seen := map[string]bool{}
	for i := range c.Apps {
		a := &c.Apps[i]
		if seen[a.ID] {
			return nil, fmt.Errorf("duplicate app id %q", a.ID)
		}
		seen[a.ID] = true
		if a.Label == "" {
			a.Label = a.ID
		}
		if appsRoot != "" {
			a.Path = filepath.Join(appsRoot, filepath.Base(a.Path))
		} else if !filepath.IsAbs(a.Path) {
			a.Path = filepath.Clean(filepath.Join(c.Dir, a.Path))
		}
	}
	if c.SiteTitle == "" {
		c.SiteTitle = "Carrybee Handbook"
	}
	return &c, nil
}
```

- [ ] **Step 5: Write** `cmd/handbook/main.go` (commands filled in by later tasks)

```go
// Command handbook builds and serves the Carrybee onboarding handbook.
package main

import (
	"flag"
	"fmt"
	"os"

	"carrybee/handbook/internal/config"
)

const usage = `usage: handbook <build|serve|check> [flags]
  -config string    path to handbook.json (default "handbook.json")
  -apps-root string directory containing hive/, aladdin/, accounts/ (overrides config paths)
  -out string       output directory for build (default "dist")
  -port int         port for serve (default 4321)`

type options struct {
	configPath, appsRoot, out string
	port                       int
}

func parse(args []string) (string, options, error) {
	if len(args) == 0 {
		return "", options{}, fmt.Errorf("missing command")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var o options
	fs.StringVar(&o.configPath, "config", "handbook.json", "")
	fs.StringVar(&o.appsRoot, "apps-root", "", "")
	fs.StringVar(&o.out, "out", "dist", "")
	fs.IntVar(&o.port, "port", 4321, "")
	if err := fs.Parse(args[1:]); err != nil {
		return "", o, err
	}
	return args[0], o, nil
}

func main() {
	cmd, o, err := parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, err := config.Load(o.configPath, o.appsRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := run(cmd, cfg, o); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd string, cfg *config.Config, o options) error {
	switch cmd {
	case "build", "serve", "check":
		return fmt.Errorf("%s: not implemented yet", cmd)
	default:
		return fmt.Errorf("unknown command %q\n%s", cmd, usage)
	}
}
```

- [ ] **Step 6: Run tests and vet**

Run: `go test ./... && go vet ./...`
Expected: PASS, no vet output.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum handbook.json cmd internal/config
git commit -m "feat: handbook module, config loader and CLI skeleton

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Module discovery and import graph

**Files:**
- Create: `internal/scan/source.go`, `internal/scan/modules.go`, `internal/scan/modules_test.go`
- Create fixture: `internal/scan/testdata/app/src/...` (files listed in Step 1)

**Interfaces:**
- Consumes: `config.App`
- Produces:
  - `StripComments(src string) string`
  - `type Module struct { Files int; Deps, UsedByModules, UsedByRoutes, APIs, Shared map[string]int }` (JSON tags `files, deps, usedByMods, usedByRoutes, apis, shared`)
  - `type Graph struct { Modules map[string]*Module; RouteImports map[string]map[string]int; Files map[string]string }` — `Files` maps path relative to `src` → comment-stripped source.
  - `BuildGraph(app config.App, routeOf func(rel []string) string) (*Graph, error)`

- [ ] **Step 1: Create the fixture tree**

```
internal/scan/testdata/app/src/modules/orders/services/api.ts
internal/scan/testdata/app/src/modules/orders/components/list.tsx
internal/scan/testdata/app/src/modules/basket/index.ts
internal/scan/testdata/app/src/modules/warehouses/index.ts
internal/scan/testdata/app/src/lib/routes/api-routes.ts
internal/scan/testdata/app/src/app/(main)/(protected)/(dashboard_layout)/(oms)/orders/page.tsx
internal/scan/testdata/app/src/app/(main)/(protected)/(dashboard_layout)/run-not-closed/page.tsx
internal/scan/testdata/app/src/lib/constants.ts
```

`modules/orders/services/api.ts`:
```ts
import restApiClient from '@/lib/api-client';
import { API_ROUTES } from '@/lib/routes/api-routes';
import { Basket } from '@/modules/basket';
// import { Ghost } from '../../ghost';
/* import { Old } from '@/modules/old'; */
export const ordersApi = { list: () => restApiClient.get(API_ROUTES.orders.list), csv: () => API_ROUTES.bulkOrderCsv };
```
`modules/orders/components/list.tsx`:
```tsx
import { ordersApi } from '../services/api';
import { Hub } from '@/modules/warehouses';
export const List = () => null;
```
`modules/basket/index.ts`:
```ts
import { List } from '@/modules/orders/components/list';
export type Basket = { id: string };
```
`modules/warehouses/index.ts`:
```ts
export type Hub = { id: number };
```
`app/(main)/(protected)/(dashboard_layout)/(oms)/orders/page.tsx`:
```tsx
import { List } from '@/modules/orders/components/list';
export default function Page() { return null; }
```
`app/(main)/(protected)/(dashboard_layout)/run-not-closed/page.tsx`:
```tsx
import { Hub } from '@/modules/warehouses';
export default function Page() { return null; }
```
`lib/routes/api-routes.ts`:
```ts
export const API_ROUTES = { orders: { list: '/orders' }, bulkOrderCsv: '/bulk' };
```
`lib/constants.ts`:
```ts
export enum TransferStatus {
  Pending = 1,
  PickupRequested = 2,
  Delivered = 15,
  // OnTheWayToOsdCentral = 38,
}
```

- [ ] **Step 2: Write the failing tests** `internal/scan/modules_test.go`

```go
package scan

import (
	"path/filepath"
	"strings"
	"testing"

	"carrybee/handbook/internal/config"
)

func fixtureApp(t *testing.T) config.App {
	t.Helper()
	p, err := filepath.Abs("testdata/app")
	if err != nil {
		t.Fatal(err)
	}
	return config.App{ID: "fixture", Path: p}
}

func TestStripComments(t *testing.T) {
	in := "a // line\nb /* block\nstill */ c\nconst u = 'http://x.io';"
	got := StripComments(in)
	if strings.Contains(got, "line") || strings.Contains(got, "block") {
		t.Fatalf("comments kept: %q", got)
	}
	if !strings.Contains(got, "'http://x.io'") {
		t.Fatalf("string with // was damaged: %q", got)
	}
}

func TestBuildGraphModulesAndEdges(t *testing.T) {
	g, err := BuildGraph(fixtureApp(t), func(rel []string) string { return "r" })
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"orders", "basket", "warehouses"} {
		if g.Modules[m] == nil {
			t.Fatalf("module %q missing", m)
		}
	}
	if _, ok := g.Modules["old"]; ok {
		t.Fatal("module from a commented import must not exist")
	}
	if g.Modules["orders"].Deps["basket"] != 1 || g.Modules["orders"].Deps["warehouses"] != 1 {
		t.Fatalf("orders deps = %v", g.Modules["orders"].Deps)
	}
	if g.Modules["orders"].UsedByModules["basket"] != 1 {
		t.Fatalf("usedBy = %v", g.Modules["orders"].UsedByModules)
	}
	if g.Modules["orders"].Shared["lib/api-client"] == 0 {
		t.Fatalf("shared = %v", g.Modules["orders"].Shared)
	}
	if g.Modules["orders"].Files != 2 {
		t.Fatalf("files = %d", g.Modules["orders"].Files)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/scan/`
Expected: FAIL — `undefined: StripComments`, `undefined: BuildGraph`

- [ ] **Step 4: Implement** `internal/scan/source.go`

```go
// Package scan reads the Carrybee Next.js frontends into structured data.
package scan

import "strings"

// StripComments removes // and /* */ comments while leaving string and template literals intact.
func StripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	var quote byte
	for i := 0; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			b.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				i++
				b.WriteByte(src[i])
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch {
		case c == '\'' || c == '"' || c == '`':
			quote = c
			b.WriteByte(c)
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				b.WriteByte('\n')
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			b.WriteString(strings.Repeat("\n", strings.Count(src[i:i+2+end], "\n")))
			i += end + 3
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
```

- [ ] **Step 5: Implement** `internal/scan/modules.go`

```go
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
		if len(rel) > 2 {
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
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/scan/ -run 'TestStripComments|TestBuildGraph' -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/scan
git commit -m "feat(scan): module discovery and import graph with comment stripping

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Routes, API domains, statuses, cycles and ScanAll

**Files:**
- Create: `internal/scan/routes.go`, `internal/scan/api.go`, `internal/scan/statuses.go`, `internal/scan/cycles.go`, `internal/scan/scan.go`, `internal/scan/scan_test.go`

**Interfaces:**
- Consumes: `BuildGraph`, `Graph`, `Module`, `config.App`, `config.Config`
- Produces:
  - `RouteNamer(app config.App) func(rel []string) string`
  - `type Route struct { Files int; Mods map[string]int }` (JSON `files`, `mods`)
  - `type Status struct { ID int; Name string }` (JSON `id`, `name`)
  - `Cycles(deps map[string]map[string]int) [][]string` (each group sorted; groups sorted by size desc then name)
  - `type AppScan struct { App, Label, Audience, Commit string; ScannedAt time.Time; Error string; Modules map[string]*Module; Routes map[string]*Route; APIDomains map[string][]string; Statuses []Status; Cycles [][]string }` (JSON camelCase; `Files` not serialised)
  - `ScanApp(app config.App) *AppScan` (never nil; sets `Error` instead of returning error)
  - `ScanAll(cfg *config.Config) map[string]*AppScan` (concurrent)

- [ ] **Step 1: Write the failing tests** `internal/scan/scan_test.go`

```go
package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"carrybee/handbook/internal/config"
)

func TestRouteNamerHive(t *testing.T) {
	name := RouteNamer(config.App{ID: "hive"})
	cases := map[string]string{
		"(main)/(protected)/(dashboard_layout)/(oms)/orders/page.tsx":       "(oms)/orders",
		"(main)/(protected)/(dashboard_layout)/run-not-closed/page.tsx":     "run-not-closed",
		"(main)/(protected)/(dashboard_layout)/dashboard/@payments/page.tsx": "dashboard",
		"page.tsx": "/",
	}
	for in, want := range cases {
		if got := name(split(in)); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestRouteNamerAladdinTwoLevels(t *testing.T) {
	name := RouteNamer(config.App{ID: "aladdin", SkipSegments: []string{"(business)", "(merchant)"}})
	cases := map[string]string{
		"(protected)/(business)/businesses/[businessId]/orders/(create-forms)/create/page.tsx": "orders/create",
		"(protected)/(business)/businesses/(global_layout)/choose/page.tsx":                     "businesses/choose",
		"(protected)/(business)/businesses/[businessId]/stores/(list)/page.tsx":                 "stores",
	}
	for in, want := range cases {
		if got := name(split(in)); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestAPIDomains(t *testing.T) {
	text := "API_ROUTES.orders.list; API_ROUTES.business.stores.create; API_ROUTES.auth"
	got := apiDomains(text, []string{"business"})
	want := map[string]int{"orders": 1, "business.stores": 1, "auth": 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestStatusesIgnoreCommentedMembers(t *testing.T) {
	src := StripComments("export enum TransferStatus {\n Pending = 1,\n // Old = 38,\n Delivered = 15,\n}\nexport enum Other { X = 9 }")
	got := parseStatuses(src)
	want := []Status{{1, "Pending"}, {15, "Delivered"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestCycles(t *testing.T) {
	deps := map[string]map[string]int{
		"a": {"b": 1}, "b": {"c": 1}, "c": {"a": 1},
		"x": {"y": 1}, "y": {"x": 1}, "z": {"a": 1},
	}
	got := Cycles(deps)
	want := [][]string{{"a", "b", "c"}, {"x", "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestScanAppFixture(t *testing.T) {
	s := ScanApp(fixtureApp(t))
	if s.Error != "" {
		t.Fatal(s.Error)
	}
	if len(s.Modules) != 3 || s.Routes["(oms)/orders"] == nil || s.Routes["run-not-closed"] == nil {
		t.Fatalf("modules=%d routes=%v", len(s.Modules), s.Routes)
	}
	if s.Modules["orders"].UsedByRoutes["(oms)/orders"] != 1 {
		t.Fatalf("usedByRoutes = %v", s.Modules["orders"].UsedByRoutes)
	}
	if got := s.APIDomains["orders"]; !reflect.DeepEqual(got, []string{"orders"}) {
		t.Fatalf("api domains = %v", s.APIDomains)
	}
	if len(s.Statuses) != 3 {
		t.Fatalf("statuses = %v", s.Statuses)
	}
	if !reflect.DeepEqual(s.Cycles, [][]string{{"basket", "orders"}}) {
		t.Fatalf("cycles = %v", s.Cycles)
	}
}

func TestScanAllMissingApp(t *testing.T) {
	cfg := &config.Config{Apps: []config.App{fixtureApp(t), {ID: "ghost", Path: filepath.Join(os.TempDir(), "no-such-app")}}}
	out := ScanAll(cfg)
	if out["fixture"].Error != "" {
		t.Fatalf("fixture failed: %s", out["fixture"].Error)
	}
	if out["ghost"] == nil || out["ghost"].Error == "" {
		t.Fatal("missing app must be returned with Error set")
	}
}

// Parity with the prototype on the real repos; skipped when they are not next to the handbook.
func TestParityWithPrototype(t *testing.T) {
	want := map[string][3]int{"hive": {76, 82, 4}, "aladdin": {18, 54, 1}, "accounts": {10, 10, 0}}
	cfg, err := config.Load("../../handbook.json", "")
	if err != nil {
		t.Skip("no handbook.json")
	}
	for _, app := range cfg.Apps {
		if _, err := os.Stat(app.Path); err != nil {
			t.Skipf("%s not present", app.ID)
		}
		s := ScanApp(app)
		got := [3]int{len(s.Modules), len(pageRoutes(s.Routes)), len(s.Cycles)}
		if got != want[app.ID] {
			t.Errorf("%s: modules/pages/cycles = %v, want %v", app.ID, got, want[app.ID])
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/scan/`
Expected: FAIL — undefined `RouteNamer`, `apiDomains`, `parseStatuses`, `Cycles`, `ScanApp`, `ScanAll`, `pageRoutes`

- [ ] **Step 3: Implement** `internal/scan/routes.go`

```go
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
```

- [ ] **Step 4: Implement** `internal/scan/api.go`, `statuses.go`, `cycles.go`

`api.go`:
```go
package scan

import "regexp"

var apiRE = regexp.MustCompile(`API_ROUTES\.(\w+)(?:\.(\w+))?`)

func apiDomains(text string, nested []string) map[string]int {
	nest := map[string]bool{}
	for _, n := range nested {
		nest[n] = true
	}
	out := map[string]int{}
	for _, m := range apiRE.FindAllStringSubmatch(text, -1) {
		d := m[1]
		if nest[d] && m[2] != "" {
			d += "." + m[2]
		}
		out[d]++
	}
	return out
}
```

`statuses.go`:
```go
package scan

import (
	"regexp"
	"strconv"
)

var (
	transferEnumRE = regexp.MustCompile(`(?s)enum\s+TransferStatus\s*\{(.*?)\}`)
	enumMemberRE   = regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9]*)\s*=\s*(\d+)`)
)

type Status struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// parseStatuses reads members of `enum TransferStatus` from comment-stripped source.
func parseStatuses(src string) []Status {
	m := transferEnumRE.FindStringSubmatch(src)
	if m == nil {
		return nil
	}
	var out []Status
	for _, mm := range enumMemberRE.FindAllStringSubmatch(m[1], -1) {
		id, _ := strconv.Atoi(mm[2])
		out = append(out, Status{ID: id, Name: mm[1]})
	}
	return out
}
```

`cycles.go`:
```go
package scan

import "sort"

// Cycles returns strongly connected groups of size > 1 (Tarjan's algorithm).
func Cycles(deps map[string]map[string]int) [][]string {
	index, low := map[string]int{}, map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var out [][]string
	next := 0
	nodes := make([]string, 0, len(deps))
	for n := range deps {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		targets := make([]string, 0, len(deps[v]))
		for w := range deps[v] {
			targets = append(targets, w)
		}
		sort.Strings(targets)
		for _, w := range targets {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var group []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				group = append(group, w)
				if w == v {
					break
				}
			}
			if len(group) > 1 {
				sort.Strings(group)
				out = append(out, group)
			}
		}
	}
	for _, n := range nodes {
		if _, seen := index[n]; !seen {
			visit(n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) > len(out[j])
		}
		return out[i][0] < out[j][0]
	})
	return out
}
```

- [ ] **Step 5: Implement** `internal/scan/scan.go`

```go
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
	Cycles     [][]string          `json:"cycles"`
	Files      map[string]string   `json:"-"`
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
		Modules: map[string]*Module{}, Routes: map[string]*Route{}, APIDomains: map[string][]string{}}
	g, err := BuildGraph(app, RouteNamer(app))
	if err != nil {
		s.Error = err.Error()
		return s
	}
	s.Commit = gitCommit(app.Path)
	s.Files = g.Files
	s.Modules = g.Modules
	for name, files := range g.RouteFiles {
		s.Routes[name] = &Route{Files: files, Mods: g.RouteImports[name]}
		if s.Routes[name].Mods == nil {
			s.Routes[name].Mods = map[string]int{}
		}
	}
	s.Routes = pageRoutes(s.Routes)
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
```

- [ ] **Step 6: Run all scan tests including parity**

Run: `go test ./internal/scan/ -v`
Expected: PASS, including `TestParityWithPrototype` (hive 76/82/4, aladdin 18/54/1, accounts 10/10/0). If parity differs, print the extra/missing route names and fix `RouteNamer` until it matches — do not change the expected numbers.

- [ ] **Step 7: Commit**

```bash
git add internal/scan
git commit -m "feat(scan): routes, API domains, statuses, cycles and concurrent ScanAll

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Curated data and term linking

**Files:**
- Create: `data/modules.json`, `data/groups.json`, `data/stories.json`, `data/terms.json`, `data/term-pins.json`
- Create: `internal/content/data.go`, `internal/content/data_test.go`, `internal/site/terms.go`, `internal/site/terms_test.go`

**Interfaces:**
- Consumes: `AppScan.Files`, `AppScan.Modules`
- Produces:
  - `type Desc struct { Title, Text string }` — JSON is a 2-element array `["title","text"]` (custom `UnmarshalJSON`/`MarshalJSON`).
  - `type Term struct { ID, Name, Cat, Def string; Aka, Apps []string; Inferred bool; Where map[string][]string }` (JSON `id,t,cat,def,aka,apps,inferred,where`)
  - `type Category struct { ID, Name string }`
  - `type StoryStep struct { Title, Actor, Text, Status string; Mods []string }`, `type StoryPhase struct { Title, When string; Steps []StoryStep }`
  - `type Data struct { Modules map[string]map[string]Desc; Groups map[string]map[string]Desc; Stories map[string][]StoryPhase; StorySub map[string]string; Categories []Category; Terms []Term; Pins map[string]map[string][]string }`
  - `content.LoadData(dir string) (*Data, error)`
  - `site.LinkTerms(terms []content.Term, scans map[string]*scan.AppScan, pins map[string]map[string][]string) []content.Term` — returns copies with `Where` set. It lives in `internal/site` (which imports both scan and content) so scan and content never import each other.

- [ ] **Step 1: Migrate curated data from BK with this one-off script** (run once; not part of the repo)

```bash
BK=../codebase-maps-backup-20261007-1303/_generator
python3 - "$BK" <<'EOF'
import json, sys, pathlib
bk = pathlib.Path(sys.argv[1]); out = pathlib.Path("data"); out.mkdir(exist_ok=True)
json.dump(json.load(open(bk/"descriptions.json")), open(out/"modules.json","w"), indent=1, ensure_ascii=False)
json.dump(json.load(open(bk/"group_descriptions.json")), open(out/"groups.json","w"), indent=1, ensure_ascii=False)
stories = {a: json.load(open(bk/f"story_{a}.json")) for a in ("hive","aladdin","accounts")}
sub = {"hive": "from building the network to paying the seller",
       "aladdin": "from a seller signing up to getting paid",
       "accounts": "from signing in to exporting invoices and reports"}
json.dump({"intro": sub, "apps": stories}, open(out/"stories.json","w"), indent=1, ensure_ascii=False)
d = json.load(open(bk/"dictionary.json"))
for t in d["terms"]:
    t.pop("where", None)
json.dump(d, open(out/"terms.json","w"), indent=1, ensure_ascii=False)
EOF
```

Then create `data/term-pins.json` by copying the `PINS` dict from `BK/link_terms.py` lines 38-56 verbatim as JSON (keys are term names, values `{app: [modules]}`).

- [ ] **Step 2: Write the failing tests** `internal/content/data_test.go`

```go
package content

import "testing"

func TestLoadDataFromRepo(t *testing.T) {
	d, err := LoadData("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Terms) != 185 || len(d.Categories) != 12 {
		t.Fatalf("terms=%d cats=%d", len(d.Terms), len(d.Categories))
	}
	if d.Modules["hive"]["delivery-run"].Title != "Delivery run" {
		t.Fatalf("desc = %+v", d.Modules["hive"]["delivery-run"])
	}
	if len(d.Stories["hive"]) != 14 || d.StorySub["aladdin"] == "" {
		t.Fatalf("stories hive=%d", len(d.Stories["hive"]))
	}
	for _, term := range d.Terms {
		if term.ID == "" {
			t.Fatalf("term %q has no id", term.Name)
		}
	}
	if d.Pins["G-Mode"]["hive"][0] != "g-mode" {
		t.Fatalf("pins = %v", d.Pins["G-Mode"])
	}
}
```

`internal/site/terms_test.go`:
```go
package site

import (
	"reflect"
	"testing"

	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
)

func TestLinkTerms(t *testing.T) {
	scans := map[string]*scan.AppScan{"hive": {Files: map[string]string{
		"modules/hub-payments/a.ts": "const cashInHand = 1; cashInHand++",
		"modules/hub-ledger/b.ts":   "type X = { cash_in_hand: number }",
		"modules/geonix/c.ts":       "const cashInHandle = 2",
	}, Modules: map[string]*scan.Module{"hub-payments": {}, "hub-ledger": {}, "geonix": {}}}}
	terms := []content.Term{
		{Name: "Cash in hand", Aka: []string{"cash_in_hand", "cashInHand"}, Apps: []string{"hive"}},
		{Name: "Pinned", Aka: []string{"zzzz"}, Apps: []string{"hive"}},
	}
	pins := map[string]map[string][]string{"Pinned": {"hive": {"geonix"}}}
	got := LinkTerms(terms, scans, pins)
	if !reflect.DeepEqual(got[0].Where["hive"], []string{"hub-payments", "hub-ledger"}) {
		t.Fatalf("where = %v (word boundaries must reject cashInHandle)", got[0].Where)
	}
	if !reflect.DeepEqual(got[1].Where["hive"], []string{"geonix"}) {
		t.Fatalf("pin not applied: %v", got[1].Where)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/content/ ./internal/site/`
Expected: FAIL — `undefined: LoadData`, `undefined: LinkTerms`

- [ ] **Step 4: Implement** `internal/content/data.go`

```go
// Package content loads hand-written Markdown and curated JSON data.
package content

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Desc struct{ Title, Text string }

func (d *Desc) UnmarshalJSON(b []byte) error {
	var pair []string
	if err := json.Unmarshal(b, &pair); err != nil || len(pair) != 2 {
		return fmt.Errorf("description must be [title, text]: %s", b)
	}
	d.Title, d.Text = pair[0], pair[1]
	return nil
}

func (d Desc) MarshalJSON() ([]byte, error) { return json.Marshal([]string{d.Title, d.Text}) }

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Term struct {
	ID       string              `json:"id"`
	Name     string              `json:"t"`
	Cat      string              `json:"cat"`
	Def      string              `json:"def"`
	Aka      []string            `json:"aka"`
	Apps     []string            `json:"apps"`
	Inferred bool                `json:"inferred,omitempty"`
	Where    map[string][]string `json:"where,omitempty"`
}

type StoryStep struct {
	Title  string   `json:"title"`
	Actor  string   `json:"actor"`
	Text   string   `json:"text"`
	Status string   `json:"status,omitempty"`
	Mods   []string `json:"mods"`
}

type StoryPhase struct {
	Title string      `json:"title"`
	When  string      `json:"when"`
	Steps []StoryStep `json:"steps"`
}

type Data struct {
	Modules    map[string]map[string]Desc
	Groups     map[string]map[string]Desc
	Stories    map[string][]StoryPhase
	StorySub   map[string]string
	Categories []Category
	Terms      []Term
	Pins       map[string]map[string][]string
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a display name into an id usable in URLs and anchors.
func Slug(s string) string { return strings.Trim(slugRE.ReplaceAllString(strings.ToLower(s), "-"), "-") }

func readJSON(dir, name string, v any) error {
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func LoadData(dir string) (*Data, error) {
	d := &Data{}
	var stories struct {
		Intro map[string]string         `json:"intro"`
		Apps  map[string][]StoryPhase   `json:"apps"`
	}
	var terms struct {
		Categories []Category `json:"categories"`
		Terms      []Term     `json:"terms"`
	}
	for _, f := range []struct {
		name string
		v    any
	}{{"modules.json", &d.Modules}, {"groups.json", &d.Groups}, {"stories.json", &stories},
		{"terms.json", &terms}, {"term-pins.json", &d.Pins}} {
		if err := readJSON(dir, f.name, f.v); err != nil {
			return nil, err
		}
	}
	d.Stories, d.StorySub = stories.Apps, stories.Intro
	d.Categories, d.Terms = terms.Categories, terms.Terms
	for i := range d.Terms {
		if d.Terms[i].ID == "" {
			d.Terms[i].ID = Slug(d.Terms[i].Name)
		}
	}
	return d, nil
}
```

- [ ] **Step 5: Implement** `internal/site/terms.go`

```go
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
)

// LinkTerms finds the modules whose source mentions each term's code names, then applies pins.
func LinkTerms(terms []content.Term, scans map[string]*scan.AppScan, pins map[string]map[string][]string) []content.Term {
	out := make([]content.Term, len(terms))
	for i, t := range terms {
		t.Where = map[string][]string{}
		for _, app := range t.Apps {
			s := scans[app]
			if s == nil || s.Files == nil {
				continue
			}
			if mods := rankModules(t.Aka, s); len(mods) > 0 {
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

func rankModules(aka []string, s *scan.AppScan) []string {
	hits := map[string]int{}
	for _, alias := range aka {
		if len(alias) < 4 {
			continue
		}
		re := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(alias) + `($|[^A-Za-z0-9_])`)
		found := map[string]int{}
		for rel, text := range s.Files {
			parts := strings.SplitN(rel, "/", 3)
			if len(parts) < 3 || parts[0] != "modules" || s.Modules[parts[1]] == nil {
				continue
			}
			if n := len(re.FindAllStringIndex(text, -1)); n > 0 {
				found[parts[1]] += n
			}
		}
		if len(found) > 0 && len(found) <= maxSpread {
			for m, n := range found {
				hits[m] += n
			}
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
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/content/ ./internal/site/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add data internal/content internal/site
git commit -m "feat: curated data migrated from prototype and term-to-module linking

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Markdown pages, shortcodes and `check`

**Files:**
- Create: `internal/content/pages.go`, `internal/content/shortcodes.go`, `internal/content/check.go`, `internal/content/pages_test.go`
- Create fixture: `internal/content/testdata/content/business/sample.md`

**Interfaces:**
- Consumes: `Data`, `Term`, goldmark
- Produces:
  - `type Page struct { Path, Section, Title, Level, Summary string; Order int; Roles []string; Body string; Refs []Ref }` — `Path` is slash path without `.md` relative to content dir (e.g. `business/courier-101`); `Body` is raw Markdown.
  - `type Ref struct { Kind, App, ID string }` — Kind is `term`, `module` or `status`.
  - `LoadPages(dir string) ([]Page, error)` (sorted by Section then Order then Title)
  - `type Resolver struct { TermName func(id string) (string, bool); ModuleDesc func(app, id string) (Desc, bool); StatusName func(id int) (string, bool); ModuleHref func(app, id string) string; TermHref func(id string) string }`
  - `Render(md string, r Resolver) (string, error)` → HTML
  - `Check(pages []Page, r Resolver) []string` → human-readable problems, empty when fine

Shortcode syntax inside Markdown (processed before goldmark):
- `{{term "cash-in-hand"}}` → `<span class="term" data-term="cash-in-hand">Cash in hand</span>`
- `{{module "hive" "delivery-run"}}` → `<a class="where hive" href="…"><b>hive</b><span>delivery-run</span></a>`
- `{{status 15}}` → `<span class="pill st ok">15 Delivered</span>` (class `ok` for 15/18/22, `bad` for 17/19/20/32, else `go`)
- Callouts: a paragraph block starting `:::note`, `:::carrybee` or `:::warning` and ending with a line `:::` becomes `<aside class="callout note">…</aside>` (inner Markdown rendered).

Front matter is a simple `---` block of `key: value` lines; `roles` is comma-separated.

- [ ] **Step 1: Create fixture** `internal/content/testdata/content/business/sample.md`

```markdown
---
title: Sample
section: business
order: 2
level: basics
roles: frontend, qa
summary: A test page.
---
Money is {{term "cash-in-hand"}} at {{module "hive" "hub-ledger"}}, final at {{status 15}}.

:::carrybee
POD means **label** here.
:::
```

- [ ] **Step 2: Write the failing tests** `internal/content/pages_test.go`

```go
package content

import (
	"strings"
	"testing"
)

func testResolver() Resolver {
	return Resolver{
		TermName:   func(id string) (string, bool) { return map[string]string{"cash-in-hand": "Cash in hand"}[id], id == "cash-in-hand" },
		ModuleDesc: func(app, id string) (Desc, bool) { return Desc{"Hub ledger", "Cash book"}, app == "hive" && id == "hub-ledger" },
		StatusName: func(id int) (string, bool) { return "Delivered", id == 15 },
		ModuleHref: func(app, id string) string { return "../apps/" + app + ".html#mod." + id },
		TermHref:   func(id string) string { return "../terms.html#t." + id },
	}
}

func TestLoadPagesFrontMatter(t *testing.T) {
	pages, err := LoadPages("testdata/content")
	if err != nil {
		t.Fatal(err)
	}
	p := pages[0]
	if p.Path != "business/sample" || p.Title != "Sample" || p.Order != 2 || p.Level != "basics" {
		t.Fatalf("%+v", p)
	}
	if strings.Join(p.Roles, ",") != "frontend,qa" {
		t.Fatalf("roles = %v", p.Roles)
	}
	if len(p.Refs) != 3 {
		t.Fatalf("refs = %v", p.Refs)
	}
}

func TestRenderShortcodesAndCallouts(t *testing.T) {
	pages, _ := LoadPages("testdata/content")
	html, err := Render(pages[0].Body, testResolver())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<span class="term" data-term="cash-in-hand">Cash in hand</span>`,
		`href="../apps/hive.html#mod.hub-ledger"`,
		`<span class="pill st ok">15 Delivered</span>`,
		`<aside class="callout carrybee">`,
		`<strong>label</strong>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s in\n%s", want, html)
		}
	}
}

func TestCheckUnknownReference(t *testing.T) {
	pages := []Page{{Path: "x/y", Refs: []Ref{{Kind: "term", ID: "nope"}, {Kind: "module", App: "hive", ID: "ghost"}, {Kind: "status", ID: "99"}}}}
	problems := Check(pages, testResolver())
	if len(problems) != 3 {
		t.Fatalf("problems = %v", problems)
	}
	if !strings.Contains(problems[0], "x/y") || !strings.Contains(problems[0], "nope") {
		t.Fatalf("problem must name file and reference: %q", problems[0])
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/content/`
Expected: FAIL — undefined `LoadPages`, `Render`, `Check`, `Resolver`

- [ ] **Step 4: Implement** `internal/content/pages.go`

```go
package content

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Ref struct{ Kind, App, ID string }

type Page struct {
	Path, Section, Title, Level, Summary string
	Order                                int
	Roles                                []string
	Body                                 string
	Refs                                 []Ref
}

var shortcodeRE = regexp.MustCompile(`\{\{\s*(term|module|status)\s+([^}]*)\}\}`)
var argRE = regexp.MustCompile(`"([^"]*)"|(\d+)`)

func shortcodeArgs(raw string) []string {
	var out []string
	for _, m := range argRE.FindAllStringSubmatch(raw, -1) {
		if m[1] != "" {
			out = append(out, m[1])
		} else {
			out = append(out, m[2])
		}
	}
	return out
}

func parseFrontMatter(src string) (map[string]string, string) {
	meta := map[string]string{}
	if !strings.HasPrefix(src, "---\n") {
		return meta, src
	}
	end := strings.Index(src[4:], "\n---")
	if end < 0 {
		return meta, src
	}
	sc := bufio.NewScanner(strings.NewReader(src[4 : 4+end]))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), ":"); ok {
			meta[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	body := strings.TrimPrefix(src[4+end+4:], "\n")
	return meta, body
}

func LoadPages(dir string) ([]Page, error) {
	var pages []Page
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		meta, body := parseFrontMatter(string(raw))
		rel, _ := filepath.Rel(dir, p)
		page := Page{Path: strings.TrimSuffix(filepath.ToSlash(rel), ".md"), Body: body,
			Title: meta["title"], Section: meta["section"], Level: meta["level"], Summary: meta["summary"]}
		page.Order, _ = strconv.Atoi(meta["order"])
		for _, r := range strings.Split(meta["roles"], ",") {
			if r = strings.TrimSpace(r); r != "" {
				page.Roles = append(page.Roles, r)
			}
		}
		for _, m := range shortcodeRE.FindAllStringSubmatch(body, -1) {
			args := shortcodeArgs(m[2])
			switch {
			case m[1] == "module" && len(args) == 2:
				page.Refs = append(page.Refs, Ref{"module", args[0], args[1]})
			case len(args) >= 1:
				page.Refs = append(page.Refs, Ref{Kind: m[1], ID: args[0]})
			}
		}
		pages = append(pages, page)
		return nil
	})
	sort.SliceStable(pages, func(i, j int) bool {
		a, b := pages[i], pages[j]
		if a.Section != b.Section {
			return a.Section < b.Section
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.Title < b.Title
	})
	return pages, err
}
```

- [ ] **Step 5: Implement** `internal/content/shortcodes.go` and `check.go`

`shortcodes.go`:
```go
package content

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

type Resolver struct {
	TermName   func(id string) (string, bool)
	ModuleDesc func(app, id string) (Desc, bool)
	StatusName func(id int) (string, bool)
	ModuleHref func(app, id string) string
	TermHref   func(id string) string
}

var md = goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(gmhtml.WithUnsafe()))

var calloutRE = regexp.MustCompile(`(?ms)^:::(note|carrybee|warning)\s*\n(.*?)\n:::\s*$`)

var goodStatus = map[int]bool{15: true, 18: true, 22: true}
var badStatus = map[int]bool{17: true, 19: true, 20: true, 32: true}

func expand(src string, r Resolver) string {
	return shortcodeRE.ReplaceAllStringFunc(src, func(m string) string {
		parts := shortcodeRE.FindStringSubmatch(m)
		args := shortcodeArgs(parts[2])
		switch parts[1] {
		case "term":
			name, ok := r.TermName(args[0])
			if !ok {
				name = args[0]
			}
			return fmt.Sprintf(`<span class="term" data-term="%s">%s</span>`, html.EscapeString(args[0]), html.EscapeString(name))
		case "module":
			if len(args) != 2 {
				return m
			}
			title := ""
			if d, ok := r.ModuleDesc(args[0], args[1]); ok {
				title = fmt.Sprintf(` title="%s"`, html.EscapeString(d.Title+": "+d.Text))
			}
			return fmt.Sprintf(`<a class="where %s" href="%s"%s><b>%s</b><span>%s</span></a>`,
				args[0], r.ModuleHref(args[0], args[1]), title, args[0], args[1])
		case "status":
			id, _ := strconv.Atoi(args[0])
			name, _ := r.StatusName(id)
			cls := "go"
			if goodStatus[id] {
				cls = "ok"
			} else if badStatus[id] {
				cls = "bad"
			}
			return fmt.Sprintf(`<span class="pill st %s">%d %s</span>`, cls, id, html.EscapeString(name))
		}
		return m
	})
}

// Render turns handbook Markdown into HTML.
func Render(src string, r Resolver) (string, error) {
	src = expand(src, r)
	var renderErr error
	src = calloutRE.ReplaceAllStringFunc(src, func(m string) string {
		parts := calloutRE.FindStringSubmatch(m)
		var inner bytes.Buffer
		if err := md.Convert([]byte(parts[2]), &inner); err != nil {
			renderErr = err
		}
		return fmt.Sprintf("<aside class=\"callout %s\">\n%s</aside>\n", parts[1], strings.TrimSpace(inner.String()))
	})
	var out bytes.Buffer
	if err := md.Convert([]byte(src), &out); err != nil {
		return "", err
	}
	return out.String(), renderErr
}
```

`check.go`:
```go
package content

import (
	"fmt"
	"strconv"
)

// Check reports references in pages that do not resolve.
func Check(pages []Page, r Resolver) []string {
	var problems []string
	for _, p := range pages {
		for _, ref := range p.Refs {
			ok := false
			switch ref.Kind {
			case "term":
				_, ok = r.TermName(ref.ID)
			case "module":
				_, ok = r.ModuleDesc(ref.App, ref.ID)
			case "status":
				id, err := strconv.Atoi(ref.ID)
				if err == nil {
					_, ok = r.StatusName(id)
				}
			}
			if !ok {
				name := ref.ID
				if ref.App != "" {
					name = ref.App + "/" + ref.ID
				}
				problems = append(problems, fmt.Sprintf("content/%s.md: unknown %s %q", p.Path, ref.Kind, name))
			}
		}
	}
	return problems
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/content/ -v && go vet ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/content
git commit -m "feat(content): Markdown pages with front matter, shortcodes, callouts and check

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Theme, layout, site builder, search and link checker

Implementer: load the `frontend-design` and `artifact-design` skill guidance before writing CSS. Read the prototype CSS in `BK/template.html` lines 1-260 and `../codebase-maps/index.html` (`<style>` block) for the brand tokens, chips, popovers and nav already designed; reuse them.

**Files:**
- Create: `theme/embed.go`, `theme/templates/layout.html`, `theme/templates/home.html`, `theme/templates/article.html`, `theme/templates/role.html`, `theme/assets/handbook.css`, `theme/assets/handbook.js`
- Create: `internal/site/build.go`, `internal/site/render.go`, `internal/site/search.go`, `internal/site/linkcheck.go`, `internal/site/build_test.go`
- Modify: `cmd/handbook/main.go` (wire `build` and `check`)

**Interfaces:**
- Consumes: `config.Config`, `scan.ScanAll`, `content.LoadData`, `content.LoadPages`, `content.Render`, `content.Check`, `site.LinkTerms`
- Produces:
  - `type Result struct { Scans map[string]*scan.AppScan; Pages int; Warnings []string; Missing []string }` — `Missing` lists `app/module` without a description and term ids without links.
  - `site.Build(cfg *config.Config, outDir string) (*Result, error)` — writes a complete site into `outDir` (creates it).
  - `site.CheckLinks(dir string) []string` — broken relative links in every `.html` under dir.
  - `site.Rel(fromPage, to string) string` — relative URL from one output path (e.g. `business/courier-101.html`) to another (e.g. `terms.html#t.x`).
  - Output tree: `index.html`, `business/*.html`, `code/*.html`, `roles/*.html`, `known-issues.html`, `terms.html`, `apps/<id>.html` (Task 7), `code/api-index.html`, `business/statuses.html` gets generated tables appended (Task 8), `assets/handbook.css`, `assets/handbook.js`, `assets/explorer.js`, `data/search.js`, `data/terms.js`, `data/sync.js`, `data/<app>.js`.
  - Data files are JS: `window.HB = window.HB || {}; HB.search = [...]` (likewise `HB.terms`, `HB.sync`, `HB.app`).

Layout contract (`layout.html`) — every page gets:
- `<html data-theme>` set from `localStorage` by an inline 3-line script in `<head>` before CSS paints.
- Top bar: brand mark + "Carrybee Handbook", search button (`⌘K` / `Ctrl K`) opening a dialog, Sync pill `#sync-pill`, theme toggle.
- Left sidebar: sections Home, Business, Apps, Code, Terms, Roles, Known issues, each listing its pages; current page has `aria-current="page"` and a yellow left rail. On ≤ 900px the sidebar is a drawer opened by a menu button.
- Main: breadcrumb, `<h1>`, level tag, rendered body, prev/next links within the section.
- Right rail (≥ 1200px, article pages only): "On this page" built from `h2` headings by JS.
- Scripts at end: `data/search.js`, `data/terms.js`, `data/sync.js`, `assets/handbook.js` (all relative via `Rel`).

`handbook.js` responsibilities: theme toggle; drawer; ⌘K search dialog over `HB.search` (group by `type`, arrow keys, Enter, Esc, highlight matches, ignore the word "carrybee" when matching); term popovers for `.term[data-term]` using `HB.terms`; module chips already link; right-rail TOC; Sync pill (local mode handled in Task 9 — here render deployed mode: text `Synced <relative time> · <app> <commit>` from `HB.sync`, click opens a panel listing every app's commit and build time and "This site rebuilds automatically on every push").

- [ ] **Step 1: Write the failing tests** `internal/site/build_test.go`

```go
package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"carrybee/handbook/internal/config"
)

func fixtureConfig(t *testing.T) *config.Config {
	t.Helper()
	app, _ := filepath.Abs("../scan/testdata/app")
	dir, _ := filepath.Abs("../..")
	return &config.Config{SiteTitle: "Test", Dir: dir, Apps: []config.App{{ID: "hive", Label: "hive", Path: app}}}
}

func TestRel(t *testing.T) {
	cases := [][3]string{
		{"index.html", "terms.html#t.x", "terms.html#t.x"},
		{"business/courier-101.html", "terms.html", "../terms.html"},
		{"business/courier-101.html", "business/statuses.html", "statuses.html"},
		{"apps/hive.html", "assets/handbook.css", "../assets/handbook.css"},
	}
	for _, c := range cases {
		if got := Rel(c[0], c[1]); got != c[2] {
			t.Errorf("Rel(%q,%q) = %q want %q", c[0], c[1], got, c[2])
		}
	}
}

func TestBuildProducesSiteWithValidLinks(t *testing.T) {
	out := t.TempDir()
	res, err := Build(fixtureConfig(t), out)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"index.html", "terms.html", "assets/handbook.css", "assets/handbook.js", "data/search.js", "data/sync.js"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if broken := CheckLinks(out); len(broken) > 0 {
		t.Fatalf("broken links: %v", broken)
	}
	if res.Pages == 0 {
		t.Fatal("no content pages rendered")
	}
}

func TestLinksAreRelative(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	abs := regexp.MustCompile(`(?:href|src)="/[^/]`)
	filepath.WalkDir(out, func(p string, d os.DirEntry, err error) error {
		if strings.HasSuffix(p, ".html") {
			b, _ := os.ReadFile(p)
			if loc := abs.Find(b); loc != nil {
				t.Errorf("%s has absolute link %s", p, loc)
			}
		}
		return nil
	})
}

func TestNoFetchInTheme(t *testing.T) {
	for _, f := range []string{"../../theme/assets/handbook.js", "../../theme/assets/explorer.js"} {
		b, err := os.ReadFile(f)
		if err != nil {
			continue // explorer.js arrives in Task 7
		}
		if strings.Contains(string(b), "fetch(") && !strings.Contains(string(b), "/api/") {
			t.Errorf("%s uses fetch for data; data must come from data/*.js so file:// works", f)
		}
	}
}

func TestSearchIndexCoversEverything(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "data/search.js"))
	for _, want := range []string{`"type":"module"`, `"title":"orders"`, `"type":"term"`, `"type":"status"`, `"type":"api"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("search index missing %s", want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/site/`
Expected: FAIL — undefined `Rel`, `Build`, `CheckLinks`

- [ ] **Step 3: Implement** `theme/embed.go`

```go
// Package theme embeds the handbook templates and static assets.
package theme

import "embed"

//go:embed templates/*.html assets/*
var FS embed.FS
```

- [ ] **Step 4: Implement** `internal/site/render.go` (templates, nav model, Rel)

```go
package site

import (
	"bytes"
	"html/template"
	"path"
	"strings"

	"carrybee/handbook/theme"
)

// Rel returns a relative URL from the page at `from` to the output path `to` (which may carry #fragment).
func Rel(from, to string) string {
	target, frag, _ := strings.Cut(to, "#")
	fromDir := path.Dir(from)
	if fromDir == "." {
		fromDir = ""
	}
	up := ""
	if fromDir != "" {
		up = strings.Repeat("../", strings.Count(fromDir, "/")+1)
	}
	rel := up + target
	if fromDir != "" && strings.HasPrefix(target, fromDir+"/") {
		rel = strings.TrimPrefix(target, fromDir+"/")
	}
	if frag != "" {
		rel += "#" + frag
	}
	return rel
}

type NavItem struct {
	Title, Href string
	Current     bool
}

type NavSection struct {
	Title string
	Items []NavItem
}

type PageView struct {
	SiteTitle   string
	Title       string
	Section     string
	Level       string
	Summary     string
	Body        template.HTML
	Nav         []NavSection
	Crumbs      []NavItem
	Prev, Next  *NavItem
	Root        string // relative prefix to site root, e.g. "../"
	Wide        bool   // maps and tables use full width
	Extra       any    // template-specific data
}

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"lower": strings.ToLower,
}).ParseFS(theme.FS, "templates/*.html"))

func renderTemplate(name string, v PageView) ([]byte, error) {
	var b bytes.Buffer
	err := templates.ExecuteTemplate(&b, name, v)
	return b.Bytes(), err
}

func rootFor(out string) string {
	n := strings.Count(out, "/")
	return strings.Repeat("../", n)
}
```

- [ ] **Step 5: Implement** `internal/site/search.go` and `linkcheck.go`

`search.go`:
```go
package site

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
)

type SearchEntry struct {
	Type  string `json:"type"`  // page | term | module | route | status | api
	Title string `json:"title"`
	Sub   string `json:"sub"`   // short context line
	Href  string `json:"href"`  // root-relative output path (JS prefixes with HB.root)
	Text  string `json:"text"`  // extra words to match
}

func searchIndex(pages []content.Page, terms []content.Term, data *content.Data, scans map[string]*scan.AppScan) []SearchEntry {
	var out []SearchEntry
	for _, p := range pages {
		out = append(out, SearchEntry{"page", p.Title, p.Summary, p.Path + ".html", p.Section})
	}
	for _, t := range terms {
		out = append(out, SearchEntry{"term", t.Name, t.Def, "terms.html#t." + t.ID, fmt.Sprint(t.Aka)})
	}
	apps := make([]string, 0, len(scans))
	for a := range scans {
		apps = append(apps, a)
	}
	sort.Strings(apps)
	seenStatus := map[int]bool{}
	for _, a := range apps {
		s := scans[a]
		for m := range s.Modules {
			d := data.Modules[a][m]
			out = append(out, SearchEntry{"module", m, a + " · " + d.Title, "apps/" + a + ".html#mod." + m, d.Text})
		}
		for r := range s.Routes {
			out = append(out, SearchEntry{"route", r, a + " page", "apps/" + a + ".html", ""})
		}
		for d, mods := range s.APIDomains {
			out = append(out, SearchEntry{"api", d, a + " · used by " + strconv.Itoa(len(mods)) + " modules", "code/api-index.html#api." + a + "." + d, fmt.Sprint(mods)})
		}
		for _, st := range s.Statuses {
			if !seenStatus[st.ID] {
				seenStatus[st.ID] = true
				out = append(out, SearchEntry{"status", strconv.Itoa(st.ID) + " " + st.Name, "order status", "business/statuses.html#s." + strconv.Itoa(st.ID), ""})
			}
		}
	}
	return out
}

func jsData(key string, v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []byte("window.HB = window.HB || {}; HB." + key + " = " + string(b) + ";\n"), nil
}
```

`linkcheck.go`:
```go
package site

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var linkRE = regexp.MustCompile(`(?:href|src)="([^"#?]+)`)

// CheckLinks returns "page -> target" for relative links that point at missing files.
func CheckLinks(dir string) []string {
	var broken []string
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, _ := os.ReadFile(p)
		for _, m := range linkRE.FindAllStringSubmatch(string(b), -1) {
			target := m[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "data:") {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(p), target)); err != nil {
				rel, _ := filepath.Rel(dir, p)
				broken = append(broken, rel+" -> "+target)
			}
		}
		return nil
	})
	return broken
}
```

- [ ] **Step 6: Implement** `internal/site/build.go`

```go
package site

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
	"carrybee/handbook/theme"
)

type Result struct {
	Scans    map[string]*scan.AppScan
	Pages    int
	Warnings []string
	Missing  []string
}

var sectionOrder = []struct{ ID, Title string }{
	{"business", "Business"}, {"code", "Code"}, {"roles", "Roles"},
}

type site struct {
	cfg    *config.Config
	out    string
	data   *content.Data
	pages  []content.Page
	terms  []content.Term
	scans  map[string]*scan.AppScan
	res    *Result
}

func Build(cfg *config.Config, outDir string) (*Result, error) {
	data, err := content.LoadData(filepath.Join(cfg.Dir, "data"))
	if err != nil {
		return nil, err
	}
	pages, err := content.LoadPages(filepath.Join(cfg.Dir, "content"))
	if err != nil {
		return nil, err
	}
	scans := scan.ScanAll(cfg)
	s := &site{cfg: cfg, out: outDir, data: data, pages: pages, scans: scans,
		res: &Result{Scans: scans}}
	s.terms = LinkTerms(data.Terms, scans, data.Pins)
	for _, step := range []func() error{s.copyAssets, s.writeData, s.writePages, s.writeTerms,
		s.writeApps, s.writeGenerated} {
		if err := step(); err != nil {
			return s.res, err
		}
	}
	s.collectMissing()
	return s.res, nil
}

func (s *site) write(rel string, b []byte) error {
	p := filepath.Join(s.out, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func (s *site) copyAssets() error {
	return fs.WalkDir(theme.FS, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := theme.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return s.write(p, b)
	})
}

func (s *site) resolver(from string) content.Resolver {
	termNames := map[string]string{}
	for _, t := range s.terms {
		termNames[t.ID] = t.Name
	}
	statuses := map[int]string{}
	for _, sc := range s.scans {
		for _, st := range sc.Statuses {
			statuses[st.ID] = st.Name
		}
	}
	return content.Resolver{
		TermName: func(id string) (string, bool) { n, ok := termNames[id]; return n, ok },
		ModuleDesc: func(app, id string) (content.Desc, bool) {
			if sc := s.scans[app]; sc == nil || sc.Modules[id] == nil {
				return content.Desc{}, false
			}
			return s.data.Modules[app][id], true
		},
		StatusName: func(id int) (string, bool) { n, ok := statuses[id]; return n, ok },
		ModuleHref: func(app, id string) string { return Rel(from, "apps/"+app+".html#mod."+id) },
		TermHref:   func(id string) string { return Rel(from, "terms.html#t."+id) },
	}
}

func (s *site) writeData() error {
	commits := map[string]string{}
	for id, sc := range s.scans {
		commits[id] = sc.Commit
	}
	sync := map[string]any{"builtAt": time.Now().UTC().Format(time.RFC3339), "commits": commits}
	files := map[string]any{
		"data/search.js": searchIndex(s.pages, s.terms, s.data, s.scans),
		"data/terms.js":  map[string]any{"categories": s.data.Categories, "terms": s.terms, "modules": s.data.Modules},
		"data/sync.js":   sync,
	}
	keys := map[string]string{"data/search.js": "search", "data/terms.js": "terms", "data/sync.js": "sync"}
	for name, v := range files {
		b, err := jsData(keys[name], v)
		if err != nil {
			return err
		}
		if err := s.write(name, b); err != nil {
			return err
		}
	}
	return nil
}

// nav builds the sidebar for the page at output path `cur`.
func (s *site) nav(cur string) []NavSection {
	secs := []NavSection{{Title: "Start", Items: []NavItem{{Title: "Home", Href: Rel(cur, "index.html"), Current: cur == "index.html"}}}}
	for _, so := range sectionOrder {
		sec := NavSection{Title: so.Title}
		for _, p := range s.pages {
			if p.Section == so.ID {
				out := p.Path + ".html"
				sec.Items = append(sec.Items, NavItem{p.Title, Rel(cur, out), out == cur})
			}
		}
		if so.ID == "code" {
			sec.Items = append(sec.Items, NavItem{"API domain index", Rel(cur, "code/api-index.html"), cur == "code/api-index.html"})
		}
		secs = append(secs, sec)
	}
	apps := NavSection{Title: "Apps"}
	for _, a := range s.cfg.Apps {
		out := "apps/" + a.ID + ".html"
		apps.Items = append(apps.Items, NavItem{a.Label, Rel(cur, out), out == cur})
	}
	secs = append(secs[:2], append([]NavSection{apps}, secs[2:]...)...)
	secs = append(secs, NavSection{Title: "Reference", Items: []NavItem{
		{"Courier terms", Rel(cur, "terms.html"), cur == "terms.html"},
		{"Known issues", Rel(cur, "known-issues.html"), cur == "known-issues.html"},
	}})
	return secs
}

func (s *site) view(cur, title string) PageView {
	return PageView{SiteTitle: s.cfg.SiteTitle, Title: title, Nav: s.nav(cur), Root: rootFor(cur)}
}

func (s *site) writePages() error {
	bySection := map[string][]content.Page{}
	for _, p := range s.pages {
		bySection[p.Section] = append(bySection[p.Section], p)
	}
	for _, p := range s.pages {
		out := p.Path + ".html"
		if p.Path == "home" {
			out = "index.html"
		}
		body, err := content.Render(p.Body, s.resolver(out))
		if err != nil {
			return fmt.Errorf("render %s: %w", p.Path, err)
		}
		v := s.view(out, p.Title)
		v.Section, v.Level, v.Summary, v.Body = p.Section, p.Level, p.Summary, template.HTML(body)
		siblings := bySection[p.Section]
		for i, sib := range siblings {
			if sib.Path == p.Path {
				if i > 0 {
					v.Prev = &NavItem{Title: siblings[i-1].Title, Href: Rel(out, siblings[i-1].Path+".html")}
				}
				if i+1 < len(siblings) {
					v.Next = &NavItem{Title: siblings[i+1].Title, Href: Rel(out, siblings[i+1].Path+".html")}
				}
			}
		}
		tpl := "article.html"
		switch {
		case p.Path == "home":
			tpl = "home.html"
			v.Extra = s.homeExtra(out)
		case p.Section == "roles":
			tpl = "role.html"
		}
		b, err := renderTemplate(tpl, v)
		if err != nil {
			return fmt.Errorf("template %s for %s: %w", tpl, p.Path, err)
		}
		if err := s.write(out, b); err != nil {
			return err
		}
		s.res.Pages++
	}
	if problems := content.Check(s.pages, s.resolver("index.html")); len(problems) > 0 {
		s.res.Warnings = append(s.res.Warnings, problems...)
	}
	return nil
}

type RoleCard struct{ ID, Title, Blurb, Href string }

func (s *site) homeExtra(cur string) map[string]any {
	var roles []RoleCard
	for _, p := range s.pages {
		if p.Section == "roles" {
			roles = append(roles, RoleCard{path.Base(p.Path), p.Title, p.Summary, Rel(cur, p.Path+".html")})
		}
	}
	counts := map[string]int{}
	for id, sc := range s.scans {
		counts[id+"Modules"] = len(sc.Modules)
		counts[id+"Pages"] = len(sc.Routes)
	}
	return map[string]any{"Roles": roles, "Counts": counts, "Terms": len(s.terms),
		"Courier101": Rel(cur, "business/courier-101.html"), "HowCarrybee": Rel(cur, "business/how-carrybee-works.html")}
}

func (s *site) collectMissing() {
	ids := make([]string, 0, len(s.scans))
	for id := range s.scans {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for m := range s.scans[id].Modules {
			if _, ok := s.data.Modules[id][m]; !ok {
				s.res.Missing = append(s.res.Missing, "module "+id+"/"+m+" has no description in data/modules.json")
			}
		}
		if s.scans[id].Error != "" {
			s.res.Warnings = append(s.res.Warnings, id+": not scanned: "+s.scans[id].Error)
		}
	}
	for _, t := range s.terms {
		if len(t.Where) == 0 {
			s.res.Missing = append(s.res.Missing, "term "+t.ID+" is not linked to any module (add a pin)")
		}
	}
	sort.Strings(s.res.Missing)
}
```

`writeTerms`, `writeApps`, `writeGenerated` are placeholders in this task (`internal/site/stubs.go`). They write simple pages so the sidebar links resolve and `CheckLinks` passes; Task 7 replaces `writeApps`, Task 8 replaces `writeTerms` and `writeGenerated` (delete each stub when replacing it):
```go
package site

import "html/template"

func (s *site) placeholder(out, title string) error {
	v := s.view(out, title)
	v.Body = template.HTML("<p>Generated in a later task.</p>")
	b, err := renderTemplate("article.html", v)
	if err != nil {
		return err
	}
	return s.write(out, b)
}

func (s *site) writeTerms() error { return s.placeholder("terms.html", "Courier terms") }

func (s *site) writeApps() error {
	for _, a := range s.cfg.Apps {
		if err := s.placeholder("apps/"+a.ID+".html", a.Label); err != nil {
			return err
		}
	}
	return nil
}

func (s *site) writeGenerated() error { return s.placeholder("code/api-index.html", "API domain index") }
```

Also create minimal content so the build has pages (Task 10 replaces the bodies): `content/home.md`, `content/business/courier-101.md`, `content/business/how-carrybee-works.md`, `content/business/statuses.md`, `content/known-issues.md`, each with front matter and one paragraph, e.g.

```markdown
---
title: Courier 101
section: business
order: 1
level: basics
roles: frontend, backend, qa, product, ops
summary: Logistics basics for people new to courier work.
---
Placeholder paragraph replaced in Task 10.
```

- [ ] **Step 7: Write the templates and assets**

`theme/templates/layout.html` defines `{{define "head"}}`, `{{define "top"}}` (top bar + sidebar + main open), `{{define "bottom"}}` (main close + scripts). Required markup hooks used by `handbook.js`: `#search-open`, `dialog#search`, `#search-input`, `#search-results`, `#sync-pill`, `#sync-panel`, `#theme-toggle`, `#menu-toggle`, `nav.sidebar`, `main#content`, `aside#toc`, `div#tip[role=tooltip]`. Inline before CSS:
```html
<script>try{var t=localStorage.getItem('hb-theme');if(t)document.documentElement.dataset.theme=t}catch(e){}</script>
```
Scripts (end of body): `{{.Root}}data/search.js`, `{{.Root}}data/terms.js`, `{{.Root}}data/sync.js`, `{{.Root}}assets/handbook.js`, plus `<script>HB.root = "{{.Root}}";</script>` before them.

`home.html`: hero "Carrybee in 60 seconds" (body from home.md), the parcel strip (inline SVG: store → FMH → hubs → LMH → buyer with a ৳ arrow back; animated dash on the path only when `prefers-reduced-motion: no-preference`), two door cards linking `.Extra.Courier101` ("New to courier? Start with Courier 101") and `.Extra.HowCarrybee` ("Already know logistics? See how Carrybee works"), five role cards from `.Extra.Roles`, and a stat row (modules/pages per app, term count) from `.Extra.Counts`.

`article.html`: breadcrumb, level tag (`basics` → "Basics", `carrybee` → "Carrybee-specific", `code` → "Code"), h1, summary lead, body, prev/next.

`role.html`: same as article with a role header band (icon letter in a yellow circle) — the Markdown body carries "What you care about", "Read in this order", and the cheat-sheet.

`handbook.css`: tokens (copy the `:root` / dark blocks from `../codebase-maps/index.html` verbatim, they are the approved brand tokens), base typography (Archivo display/body, JetBrains Mono for code — Google Fonts link in head), layout grid (`240px sidebar | minmax(0,72ch) content | 220px toc`), components: `.term`, `#tip`, `.where`, `.pill.st.ok|go|bad`, `.callout.note|carrybee|warning`, `.chip`, `.card`, tables, code blocks, `dialog#search`, sync pill and panel, drawer, focus-visible rings, reduced motion.

- [ ] **Step 8: Wire `build` and `check`** in `cmd/handbook/main.go` `run`:

```go
	case "build":
		res, err := site.Build(cfg, o.out)
		if err != nil {
			return err
		}
		printResult(res)
		return nil
	case "check":
		tmp, err := os.MkdirTemp("", "handbook-check-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		res, err := site.Build(cfg, tmp)
		if err != nil {
			return err
		}
		printResult(res)
		problems := append(append([]string{}, res.Warnings...), res.Missing...)
		problems = append(problems, site.CheckLinks(tmp)...)
		if len(problems) > 0 {
			return fmt.Errorf("check failed: %d problem(s)", len(problems))
		}
		fmt.Println("check passed")
		return nil
```
with
```go
func printResult(res *site.Result) {
	for id, s := range res.Scans {
		status := "ok"
		if s.Error != "" {
			status = "NOT SCANNED: " + s.Error
		}
		fmt.Printf("%-9s %3d modules %3d pages %2d cycles  %s  %s\n", id, len(s.Modules), len(s.Routes), len(s.Cycles), s.Commit, status)
	}
	fmt.Printf("pages rendered: %d\n", res.Pages)
	for _, w := range res.Warnings {
		fmt.Println("warning:", w)
	}
	for _, m := range res.Missing {
		fmt.Println("needs description:", m)
	}
}
```

- [ ] **Step 9: Run tests and a real build**

Run: `go test ./... && go vet ./... && go run ./cmd/handbook build`
Expected: tests PASS; build prints hive 76/82/4, aladdin 18/54/1, accounts 10/10/0 and writes `dist/`.

- [ ] **Step 10: Visual check** — headless Chrome screenshots of `dist/index.html` and `dist/business/courier-101.html` in light and dark at 1300px and 400px:

```bash
C="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
for page in index business/courier-101; do for w in 1300 400; do
  "$C" --headless=new --disable-gpu --window-size=$w,1400 --virtual-time-budget=3000 \
    --screenshot=/tmp/hb-$(echo $page | tr / -)-$w.png "file://$PWD/dist/$page.html"
done; done
```
Fix anything clipped, overlapping, or unreadable; one pass.

- [ ] **Step 11: Commit**

```bash
git add theme internal/site cmd content
git commit -m "feat(site): branded layout, home, articles, search index, link checker, build and check commands

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: App map pages (explorer port)

**Files:**
- Create: `theme/assets/explorer.js`, `theme/templates/app.html`, `internal/site/appmap.go`, `internal/site/appmap_test.go`
- Modify: `internal/site/stubs.go` (remove `writeApps`)

**Interfaces:**
- Consumes: `AppScan`, `Data.Modules`, `Data.Groups`, `Data.Stories`, `Data.StorySub`, linked `[]content.Term`
- Produces: `dist/apps/<id>.html` and `dist/data/<id>.js` defining
  `HB.app = { id, label, audience, commit, error, mods, routes, cycles, desc, groups, story, storySub, termLinks }`
  where `mods`/`routes`/`cycles` follow the prototype `DATA` shape: `mods[name] = {files, deps, usedByMods, usedByRoutes, apis, shared}`, `routes[name] = {files, mods}`, `termLinks[module] = [[termId, termName], …]`.

Port instructions (no behaviour changes except those listed):
1. Take the full `<script>` body of `BK/template.html` (lines 264-end) into `theme/assets/explorer.js`.
2. Replace the placeholder constants with reads from `HB.app`:
   `const DATA = {mods: HB.app.mods, routes: HB.app.routes, cycles: HB.app.cycles}; const DESC = HB.app.desc; const STORY = HB.app.story; const GROUPS = HB.app.groups; const TERMLINKS = HB.app.termLinks;`
   and `'__DEFAULT__'` with the first story-step page or the first route; `'__APP__-map'` storage key with `'hb-map-' + HB.app.id`; `__STORYSUB__` with `HB.app.storySub`.
3. Remove the explorer's own top nav and tooltip element creation that duplicate the layout (`#tip` comes from the layout; reuse it).
4. Term links: `terms.html#t.<id>` becomes `HB.root + 'terms.html#t.' + id`.
5. Keep d3 for the network view, loaded only on app pages: `<script src="https://cdnjs.cloudflare.com/ajax/libs/d3/7.9.0/d3.min.js"></script>`.
6. Move the explorer CSS (BK/template.html `<style>` minus tokens/body/nav rules) into `theme/assets/explorer.css`, linked only from `app.html`.
7. If `HB.app.error` is set, render a "Not scanned: <error>" callout instead of the explorer.

`app.html` adds an overview header above the explorer: who uses the app (from `audience`), module/page/cycle counts, commit, and a short "Key folders" list (`src/app`, `src/modules`, `src/lib/routes/app-routes.ts`, `src/lib/routes/api-routes.ts`, `src/proxy.ts`, `auth.ts`).

- [ ] **Step 1: Write the failing test** `internal/site/appmap_test.go`

```go
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppPagesAndData(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "apps/hive.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`../data/hive.js`, `../assets/explorer.js`, `id="main"`} {
		if !strings.Contains(string(page), want) {
			t.Errorf("app page missing %s", want)
		}
	}
	data, _ := os.ReadFile(filepath.Join(out, "data/hive.js"))
	for _, want := range []string{`HB.app = `, `"mods":{`, `"orders":{`, `"cycles":[["basket","orders"]]`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("app data missing %s", want)
		}
	}
	if broken := CheckLinks(out); len(broken) > 0 {
		t.Fatalf("broken: %v", broken)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/site/ -run TestAppPagesAndData`
Expected: FAIL (stub writes no data file)

- [ ] **Step 3: Implement** `internal/site/appmap.go`

```go
package site

import (
	"sort"

	"carrybee/handbook/internal/content"
)

func (s *site) writeApps() error {
	for _, app := range s.cfg.Apps {
		sc := s.scans[app.ID]
		termLinks := map[string][][2]string{}
		for _, t := range s.terms {
			for _, m := range t.Where[app.ID] {
				termLinks[m] = append(termLinks[m], [2]string{t.ID, t.Name})
			}
		}
		for m := range termLinks {
			sort.Slice(termLinks[m], func(i, j int) bool { return termLinks[m][i][1] < termLinks[m][j][1] })
		}
		story := s.data.Stories[app.ID]
		if story == nil {
			story = []content.StoryPhase{}
		}
		payload := map[string]any{
			"id": app.ID, "label": app.Label, "audience": app.Audience, "commit": sc.Commit, "error": sc.Error,
			"mods": sc.Modules, "routes": sc.Routes, "cycles": sc.Cycles,
			"desc": s.data.Modules[app.ID], "groups": s.data.Groups[app.ID],
			"story": story, "storySub": s.data.StorySub[app.ID], "termLinks": termLinks,
		}
		b, err := jsData("app", payload)
		if err != nil {
			return err
		}
		if err := s.write("data/"+app.ID+".js", b); err != nil {
			return err
		}
		out := "apps/" + app.ID + ".html"
		v := s.view(out, app.Label)
		v.Section, v.Wide = "apps", true
		v.Extra = map[string]any{"App": app, "Scan": sc, "Data": Rel(out, "data/"+app.ID+".js")}
		page, err := renderTemplate("app.html", v)
		if err != nil {
			return err
		}
		if err := s.write(out, page); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Port `explorer.js`, `explorer.css`, write `app.html`** per the port instructions above.

- [ ] **Step 5: Run tests, build, and verify in a browser**

Run: `go test ./internal/site/ && go run ./cmd/handbook build`
Then headless check that the explorer boots and deep links work:
```bash
C="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
"$C" --headless=new --disable-gpu --virtual-time-budget=4000 --dump-dom "file://$PWD/dist/apps/hive.html#mod.hub-payments" | grep -o 'Module · hub-payments' | head -1
```
Expected: prints `Module · hub-payments`. Screenshot the A-to-Z view and Flow view once each; fix layout issues in one pass.

- [ ] **Step 6: Commit**

```bash
git add theme internal/site
git commit -m "feat(site): app map pages with ported explorer (A-to-Z, flow, network)

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Generated pages — terms, API domain index, statuses tables

**Files:**
- Create: `internal/site/generated.go`, `internal/site/generated_test.go`, `theme/templates/terms.html`, `theme/templates/apiindex.html`
- Modify: `internal/site/stubs.go` (remove `writeTerms`, `writeGenerated`; delete the file if empty)

**Interfaces:**
- Consumes: linked terms, `Data.Categories`, `AppScan.APIDomains`, `AppScan.Modules[*].UsedByRoutes`, `AppScan.Statuses`
- Produces: `dist/terms.html` (port of `../codebase-maps/terms.html` page script: search with word-start matching and brand-word skip, category chips, "Where it lives" links to `apps/<app>.html#mod.<m>`, `#t.<id>` deep link + flash); `dist/code/api-index.html` (one table per app: API domain → modules → pages, each row has `id="api.<app>.<domain>"`); `business/statuses.html` gets an appended generated table (`id="s.<n>"` rows: number, enum name, phase — pickup/first mile/middle mile/last mile/return/reverse/final, and the matching term link).

- [ ] **Step 1: Write the failing test** `internal/site/generated_test.go`

```go
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedPages(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(fixtureConfig(t), out); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join(out, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if api := read("code/api-index.html"); !strings.Contains(api, `id="api.hive.orders"`) || !strings.Contains(api, "(oms)/orders") {
		t.Errorf("api index missing orders row with its page")
	}
	if st := read("business/statuses.html"); !strings.Contains(st, `id="s.15"`) || !strings.Contains(st, "Delivered") {
		t.Errorf("statuses table missing 15 Delivered")
	}
	if terms := read("terms.html"); !strings.Contains(terms, "data/terms.js") {
		t.Errorf("terms page must load data/terms.js")
	}
}

func TestStatusPhase(t *testing.T) {
	cases := map[int]string{2: "First mile", 9: "Middle mile", 14: "Last mile", 15: "Final", 25: "Return",
		43: "Reverse pickup", 38: "Middle mile", 53: "Change request"}
	for id, want := range cases {
		if got := statusPhase(id); got != want {
			t.Errorf("statusPhase(%d) = %q want %q", id, got, want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/site/ -run 'TestGeneratedPages|TestStatusPhase'`
Expected: FAIL — `undefined: statusPhase`

- [ ] **Step 3: Implement** `internal/site/generated.go`

```go
package site

import (
	"fmt"
	"html/template"
	"sort"
	"strings"

	"carrybee/handbook/internal/content"
)

func statusPhase(id int) string {
	switch {
	case id >= 1 && id <= 7:
		return "First mile"
	case (id >= 8 && id <= 13) || id == 35 || id == 38 || id == 39:
		return "Middle mile"
	case id == 14 || id == 16:
		return "Last mile"
	case id == 15 || id == 18 || id == 19 || id == 20 || id == 21 || id == 22 || id == 32 || id == 33:
		return "Final"
	case (id >= 23 && id <= 31) || id == 34 || id == 40 || id == 41 || id == 17:
		return "Return"
	case id >= 42 && id <= 45:
		return "Reverse pickup"
	case id == 36 || id == 37:
		return "Recovery"
	case id == 53:
		return "Change request"
	}
	return "Other"
}

func (s *site) writeTerms() error {
	v := s.view("terms.html", "Courier terms")
	v.Section, v.Wide = "reference", true
	b, err := renderTemplate("terms.html", v)
	if err != nil {
		return err
	}
	return s.write("terms.html", b)
}

type apiRow struct {
	Domain, Anchor string
	Modules        []string
	Pages          []string
}

func (s *site) writeGenerated() error {
	// API domain index
	type appTable struct {
		App  string
		Rows []apiRow
	}
	var tables []appTable
	for _, app := range s.cfg.Apps {
		sc := s.scans[app.ID]
		t := appTable{App: app.ID}
		for d, mods := range sc.APIDomains {
			pages := map[string]bool{}
			for _, m := range mods {
				for r := range sc.Modules[m].UsedByRoutes {
					pages[r] = true
				}
			}
			row := apiRow{Domain: d, Anchor: "api." + app.ID + "." + d, Modules: mods}
			for p := range pages {
				row.Pages = append(row.Pages, p)
			}
			sort.Strings(row.Pages)
			t.Rows = append(t.Rows, row)
		}
		sort.Slice(t.Rows, func(i, j int) bool { return t.Rows[i].Domain < t.Rows[j].Domain })
		tables = append(tables, t)
	}
	v := s.view("code/api-index.html", "API domain index")
	v.Section, v.Wide = "code", true
	v.Extra = map[string]any{"Tables": tables, "AppsRoot": Rel("code/api-index.html", "apps/")}
	b, err := renderTemplate("apiindex.html", v)
	if err != nil {
		return err
	}
	if err := s.write("code/api-index.html", b); err != nil {
		return err
	}
	return s.appendStatusTable()
}

// appendStatusTable re-renders business/statuses.html with a generated table after the hand-written body.
func (s *site) appendStatusTable() error {
	var page *content.Page
	for i := range s.pages {
		if s.pages[i].Path == "business/statuses" {
			page = &s.pages[i]
		}
	}
	if page == nil {
		return fmt.Errorf("content/business/statuses.md is required")
	}
	seen := map[int]string{}
	for _, sc := range s.scans {
		for _, st := range sc.Statuses {
			seen[st.ID] = st.Name
		}
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var rows strings.Builder
	rows.WriteString(`<h2 id="all-statuses">Every status in code</h2><div class="table-wrap"><table class="status-table"><thead><tr><th>#</th><th>Name in code</th><th>Phase</th></tr></thead><tbody>`)
	for _, id := range ids {
		fmt.Fprintf(&rows, `<tr id="s.%d"><td class="num">%d</td><td><code>%s</code></td><td>%s</td></tr>`,
			id, id, template.HTMLEscapeString(seen[id]), statusPhase(id))
	}
	rows.WriteString(`</tbody></table></div>`)
	out := "business/statuses.html"
	body, err := content.Render(page.Body, s.resolver(out))
	if err != nil {
		return err
	}
	v := s.view(out, page.Title)
	v.Section, v.Level, v.Summary = page.Section, page.Level, page.Summary
	v.Body = template.HTML(body + rows.String())
	b, err := renderTemplate("article.html", v)
	if err != nil {
		return err
	}
	return s.write(out, b)
}
```

- [ ] **Step 4: Write `terms.html` and `apiindex.html` templates**; port the terms page script from `../codebase-maps/terms.html` into the template (it reads `HB.terms`), adapting links to `HB.root + 'apps/' + app + '.html#mod.' + m`.

- [ ] **Step 5: Run tests and build**

Run: `go test ./... && go run ./cmd/handbook build`
Expected: PASS; `dist/code/api-index.html` lists hive's 70 domains.

- [ ] **Step 6: Commit**

```bash
git add internal/site theme
git commit -m "feat(site): generated terms page, API domain index and status table

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Local server and Sync

**Files:**
- Create: `internal/server/server.go`, `internal/server/diff.go`, `internal/server/server_test.go`
- Modify: `cmd/handbook/main.go` (wire `serve`), `theme/assets/handbook.js` (local Sync mode)

**Interfaces:**
- Consumes: `site.Build`, `site.Result`, `scan.AppScan`, `config.Config`
- Produces:
  - `type Diff struct { App string; AddedModules, RemovedModules, AddedPages, RemovedPages, AddedAPIs, RemovedAPIs []string }`
  - `func DiffScans(before, after map[string]*scan.AppScan) []Diff` (only apps with changes; slices sorted)
  - `type Server struct{…}`; `func New(cfg *config.Config, dist string, build BuildFunc) *Server`; `type BuildFunc func(cfg *config.Config, out string) (*site.Result, error)`; `func (s *Server) Handler() http.Handler`; `func (s *Server) ListenAndServe(port int) error`
  - `GET /api/status` → `{"mode":"local","builtAt":"…","commits":{…},"syncing":false}`
  - `POST /api/sync` → 200 `{"ok":true,"diff":[…],"missing":[…],"warnings":[…],"seconds":1.2}`; 409 `{"ok":false,"error":"sync already running"}`; 500 `{"ok":false,"error":"…"}` with old site untouched.

- [ ] **Step 1: Write the failing tests** `internal/server/server_test.go`

```go
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
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/server/`
Expected: FAIL — undefined `New`, `DiffScans`, `BuildFunc`, `Diff`

- [ ] **Step 3: Implement** `internal/server/diff.go`

```go
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
	var out []Diff
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
```

- [ ] **Step 4: Implement** `internal/server/server.go`

```go
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
```

- [ ] **Step 5: Wire `serve`** in `cmd/handbook/main.go`:

```go
	case "serve":
		abs, err := filepath.Abs(o.out)
		if err != nil {
			return err
		}
		return server.New(cfg, abs, site.Build).ListenAndServe(o.port)
```

- [ ] **Step 6: Local Sync in `handbook.js`** — on load, if `location.protocol` starts with `http`, `fetch(HB.root + 'api/status')`; when it answers with `mode: "local"`, the pill becomes a button "Sync". Click → disable, show spinner text "Syncing…", `POST api/sync`; on `ok` show a toast summarising the diff (e.g. "hive: +2 modules, +1 page · 3 need descriptions"), then `location.reload()` after 1.5 s; on 409 toast "A sync is already running"; on error toast the message and keep the page. (These two `fetch` calls are the only ones allowed; they hit `/api/`, which `TestNoFetchInTheme` permits.)

- [ ] **Step 7: Run tests and a live check**

Run: `go test ./... && go vet ./...`
Then:
```bash
go run ./cmd/handbook serve --port 4321 &
sleep 3; curl -s localhost:4321/api/status; curl -s -X POST localhost:4321/api/sync | head -c 300; kill %1
```
Expected: status JSON with `"mode":"local"`; sync JSON with `"ok":true`.

- [ ] **Step 8: Commit**

```bash
git add internal/server cmd theme
git commit -m "feat(server): local server with atomic Sync rebuild and diff

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: Write the handbook content

Implementer: load the `humanizer` skill guidance for tone. Plain, direct sentences; no marketing language. Use shortcodes wherever a term, module or status is mentioned for the first time on a page. Run `go run ./cmd/handbook check` after each file; it must pass at the end.

Sources of facts (all verified earlier, use them, do not invent):
- `../codebase-maps/index.html` — business, apps, A-to-Z journey, money flow (hub deposit formula: payable = sum collected − (money lost + cash in hand + withdrawn expenses) + carry), network, statuses, glossary, where to start, known issues.
- `data/terms.json`, `data/stories.json`, `data/modules.json`.
- `../aladdin/CLAUDE.md`, `../hive/architecture.md`, `../accounts/AGENTS.md`.
- Login per app: aladdin phone + password (OTP at signup, impersonate); hive backend-brokered SSO + hub selection (impersonate); accounts phone + OTP, permissions `invoice_and_billing.revenue_recognition.view` and `invoice_and_billing.legal_panel.view`.

**Files (each with front matter `title, section, order, level, roles, summary`):**

| File | section/order/level | Must contain |
|---|---|---|
| `content/home.md` | home/0/– | "Carrybee in 60 seconds": 4 short paragraphs (what Carrybee is, the three apps and who uses them, how a parcel moves, how money moves). |
| `content/business/courier-101.md` | business/1/basics | Generic logistics for newcomers: first/middle/last mile, hubs and sorting, runs and riders, COD, returns and reverse logistics, why statuses exist. Ends with `:::note` "Know this already? Go to How Carrybee works". |
| `content/business/how-carrybee-works.md` | business/2/carrybee | Business model; who uses which app; the 13-step A-to-Z journey (from index.html) with `{{status}}` and `{{module}}` shortcodes; money flow; network & geography; `:::carrybee` callout "If you already know courier" listing: POD = label, QC = approval review, urgent approval, PTN racks, hub deposit formula, ISD/OSD pricing, paisa amounts, sorting window 18:00→06:00, statuses 38–41 sub-sort / 42–45 reverse pickup. |
| `content/business/statuses.md` | business/3/carrybee | How to read statuses (numeric vs text, final statuses, groups/tabs) — generated table appends below. |
| `content/code/architecture.md` | code/1/code | Next 16 App Router, route groups and parallel routes, `src/modules/<feature>` anatomy (components/hooks/services/types/utils/validations), `app-routes.ts` / `api-routes.ts`, auth.ts + proxy.ts + permission gates, React Query, react-hook-form + Zod, shadcn/Tailwind v4, Centrifugo, OneSignal; per-app differences table. |
| `content/code/add-a-feature.md` | code/2/code | Step-by-step: add route in app-routes, page under src/app, module folder, API in api-routes + services, query keys, permission gate, i18n (aladdin), checks: `pnpm exec tsc --noEmit`, `pnpm lint`, `pnpm build`. |
| `content/code/conventions-and-gotchas.md` | code/3/code | Conventions (pnpm only, commitlint, husky full build on commit in aladdin, Sonar rules), gotchas (module isolation not enforced, circular import groups per app, no tests, Next 16 docs in node_modules, `transferStatus` old numbering in comments). |
| `content/roles/frontend.md` | roles/1/code | What you care about; read in this order (6 links); cheat sheet: commands, module anatomy, where API calls live, how permissions gate UI. |
| `content/roles/backend.md` | roles/2/code | API domain index link; status enums; money fields and formulas; paisa; which frontend calls which domain. |
| `content/roles/qa.md` | roles/3/carrybee | E2E journeys to test: happy path (signup → order → pickup → delivery → invoice → payout), failure paths (return, partial, exchange, reverse pickup, lost, urgent approval, no-entry, force close); roles/permissions to log in as; statuses to verify; link to known issues. |
| `content/roles/product.md` | roles/4/carrybee | Users per app and their jobs; journeys (A-to-Z links per app); exception flows and their business cost; metrics (delivery success, return/hold rate, aging). |
| `content/roles/ops.md` | roles/5/basics | "Where in hive do I do X" table (≥ 20 rows: task → hive page/module chip), plus terms and statuses links. |
| `content/known-issues.md` | reference/1/code | Each issue: what, where (file:line), impact, suggested fix: hive README AUTH_SECRET; hive constants.ts:15 ACCESS_TOKEN JWT; aladdin rejectUrgentApprovalOrder uses approve URL; accounts consignment & merchant reports share endpoint; aladdin APP_ROUTES without pages; circular import groups; module boundaries rule disabled; no tests. |

- [ ] **Step 1:** Write `home.md`, the three business pages and `statuses.md`; run `go run ./cmd/handbook check` — fix every unknown reference.
- [ ] **Step 2:** Write the three code pages; run check.
- [ ] **Step 3:** Write the five role pages and `known-issues.md`; run check.
- [ ] **Step 4:** Build and read every page once in the browser at 1300px (headless screenshots of each), fix copy that overflows or reads badly.
- [ ] **Step 5:** Commit

```bash
git add content
git commit -m "docs: handbook content for business, code, roles and known issues

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 11: Deployment, README and final QA

**Files:**
- Create: `.gitlab-ci.yml`, `Dockerfile`, `.dockerignore`, `README.md`

**Interfaces:**
- Consumes: the `handbook` CLI.
- Produces: CI pipeline (test → check → build → pages), Docker image serving `dist/`.

- [ ] **Step 1: Write** `.gitlab-ci.yml`

```yaml
stages: [test, build, deploy]

variables:
  GO_VERSION: "1.24"
  APPS_ROOT: "$CI_PROJECT_DIR/.apps"
  # Read-only token to clone the three app repos; set as a masked CI/CD variable, never committed.
  # APPS_CLONE_TOKEN: set in GitLab > Settings > CI/CD > Variables
  HIVE_REPO: "gitlab.com/carrybee/hive.git"
  ALADDIN_REPO: "gitlab.com/carrybee/aladdin.git"
  ACCOUNTS_REPO: "gitlab.com/carrybee/accounts.git"
  APPS_BRANCH: "deployment-prod"

.go:
  image: golang:${GO_VERSION}
  cache:
    key: go-mod
    paths: [.go/pkg/mod]
  variables:
    GOPATH: "$CI_PROJECT_DIR/.go"

.clone_apps: &clone_apps
  - mkdir -p "$APPS_ROOT"
  - for r in "$HIVE_REPO" "$ALADDIN_REPO" "$ACCOUNTS_REPO"; do
      git clone --depth 1 --branch "$APPS_BRANCH" "https://oauth2:${APPS_CLONE_TOKEN}@${r}" "$APPS_ROOT/$(basename "$r" .git)";
    done

test:
  extends: .go
  stage: test
  script:
    - go vet ./...
    - go test ./...

build:
  extends: .go
  stage: build
  script:
    - *clone_apps
    - go run ./cmd/handbook check --apps-root "$APPS_ROOT"
    - go run ./cmd/handbook build --apps-root "$APPS_ROOT" --out public
  artifacts:
    paths: [public]
    expire_in: 1 week

pages:
  stage: deploy
  needs: [build]
  script: ["echo deploying public/"]
  artifacts:
    paths: [public]
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_PIPELINE_SOURCE == "trigger"
    - if: $CI_PIPELINE_SOURCE == "web"
```

The repo URLs are placeholders for the team to confirm; README states this explicitly.

- [ ] **Step 2: Write** `Dockerfile` and `.dockerignore`

```dockerfile
# Stage 1: build the handbook binary and the static site.
# Build context must contain the handbook and, under ./.apps, the hive/aladdin/accounts sources.
FROM golang:1.24-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/handbook ./cmd/handbook \
 && /out/handbook build --apps-root /src/.apps --out /out/dist

# Stage 2a (default): static site behind nginx.
FROM nginx:1.27-alpine AS static
COPY --from=build /out/dist /usr/share/nginx/html
EXPOSE 80

# Stage 2b: the Go server, so Sync also works when hosted (`docker build --target live`).
FROM alpine:3.20 AS live
RUN apk add --no-cache git
COPY --from=build /out/handbook /usr/local/bin/handbook
WORKDIR /handbook
COPY handbook.json ./
COPY data ./data
COPY content ./content
EXPOSE 4321
ENTRYPOINT ["handbook", "serve", "--apps-root", "/apps", "--port", "4321"]
```

Note for `live`: the server binds 127.0.0.1 per the constraints; add a `--host` flag only if the team decides to expose Sync beyond localhost (out of scope now) — README documents running `live` with `--network host` for local use.

`.dockerignore`:
```
dist
.dist-*
.go
```

- [ ] **Step 3: Write** `README.md` — what it is; requirements (Go 1.24); quick start (`go run ./cmd/handbook serve` → http://127.0.0.1:4321, Sync button); commands; folder map; how to edit content (Markdown front matter, shortcodes, callouts); how to add a term/description/pin; how Sync works; CI pipeline setup (variables `APPS_CLONE_TOKEN`, repo URLs, triggers from the app repos via pipeline trigger tokens); Docker usage.

- [ ] **Step 4: Final verification**

```bash
go vet ./... && go test ./... && go run ./cmd/handbook check && go run ./cmd/handbook build
```
Expected: all pass; `check passed`.

Headless QA: screenshots in light and dark at 1300px and 400px of `index.html`, `roles/qa.html`, `apps/hive.html`, `terms.html`, `code/api-index.html`; confirm search (⌘K) returns results for "carry", "LMH", "delivery-run", "15" via `--dump-dom` with a scripted query, and that `file://` opening works (no console errors about blocked fetch).

- [ ] **Step 5: Commit**

```bash
git add .gitlab-ci.yml Dockerfile .dockerignore README.md
git commit -m "chore: GitLab Pages pipeline, Docker images and README

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

## Self-review notes

- Spec §1 criteria → Tasks 6 (role cards, two clicks), 10 (101 vs Carrybee-specific), 6+8 (search covers modules/pages/api/status/terms), 9 (Sync, no content writes), 11 (deploy), 6 (brand, themes, phone, a11y).
- Spec §5.1 scanner rules → Tasks 2-3 with tests; term linking → Task 4.
- Spec §5.2 shortcodes/callouts/check → Task 5. Spec §5.3 renderer/search/sync.json → Tasks 6-8. Spec §5.4 server → Task 9.
- Spec §6 Sync states (local / deployed / needs description) → Tasks 6 (deployed pill), 9 (local + missing list).
- Spec §8 content → Task 10. §9 → Task 11. §10 errors → Tasks 3 (missing app), 5 (unknown refs), 9 (failed sync). §11 tests → each task.
- Deviation from spec, recorded: browser data ships as `data/*.js` instead of JSON so `file://` works; `LinkTerms` lives in `internal/site` to avoid a scan↔content import cycle.
