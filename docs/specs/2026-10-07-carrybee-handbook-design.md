# Carrybee Handbook — design spec

Date: 2026-10-07 · Status: draft for review · Owner: frontend team

## 1. Purpose

A single onboarding and reference site — the "Carrybee handbook" — that lets a new employee understand
the **business** (how a parcel and its money move through Carrybee) and the **code structure** of the three
frontends (`hive`, `aladdin`, `accounts`) without asking around.

It replaces the prototype in `new-work/codebase-maps/` (kept untouched in
`new-work/codebase-maps-backup-20261007-1303/`, including the Python generators in `_generator/`).

### Success criteria

1. A new hire can pick their role on the home page and reach every page relevant to them within two clicks.
2. Someone new to courier work can learn the basics (Courier 101) before Carrybee specifics; someone who
   knows logistics can skip straight to what is Carrybee-specific.
3. Every module, page, API domain, status and courier term found in the three codebases appears in the
   handbook and is reachable through search.
4. Clicking **Sync** while running locally refreshes all generated data from the current code and reports
   what changed. Hand-written content is never overwritten.
5. `dist/` deploys unchanged to any static host; a GitLab CI job and a Dockerfile are provided.
6. Carrybee brand colours, light and dark themes, works at phone width, keyboard accessible.

### Out of scope

- Editing content through the browser (content is edited as Markdown/JSON files in git).
- Backend code, the agent mobile app and the Carrybee API itself (not in these repos).
- User accounts or analytics on the handbook.
- A guided course with enforced order or progress tracking (explicitly declined; reference handbook only).

## 2. Audiences

| Audience | What they need first |
|---|---|
| New to courier domain | Courier 101: miles, hubs, runs, COD, returns, in plain words |
| Already knows logistics | "If you already know courier": where Carrybee's words and rules differ |
| Frontend developer | App architecture, module anatomy, conventions, checks, where to start |
| Backend developer | API domain index (who calls what), status enums, money formulas, key fields |
| QA / tester | End-to-end journeys (happy and failure paths), roles/permissions, statuses, known bugs |
| Product / design / PM | User types per app, journeys, exception flows, metrics |
| Ops / support / non-tech | "Where in hive do I do X", terms, statuses |

## 3. Information architecture

```
Home                      Carrybee in 60 seconds · role cards · "New to courier?" / "Know logistics?" · search
Business
  Courier 101             tag: Basics — skip if you know logistics
  How Carrybee works      business model, three apps & users, A-to-Z parcel journey across apps,
                          money flow, network & geography, "If you already know courier" callout
  Statuses                order lifecycle state diagram + run, basket, invoice, hub-processing statuses
Apps
  hive / aladdin / accounts   overview header (users, login, key folders, commands)
                              + interactive map: A-to-Z lifecycle · Flow · Whole network
Code
  Shared architecture     Next 16 App Router, module pattern, app-routes/api-routes, auth + proxy +
                          permissions, React Query, forms, conventions, how to add a feature, gotchas
  API domain index        generated: API_ROUTES domain → app → modules → pages
Terms                     Courier Terms (185+), linked both ways with modules
Roles                     Frontend · Backend · QA · Product/Design · Ops/Support
Known issues              security findings and suspected bugs, each with file:line
```

Every page carries level tags where useful: **Basics**, **Carrybee-specific**, **Code**.

## 4. Repository layout

```
carrybee-handbook/
  cmd/handbook/main.go         CLI entry: build | serve | check
  internal/scan/               scanners for the three frontends
  internal/content/            loads Markdown (content/) and curated JSON (data/)
  internal/site/               renders pages, writes dist/, builds the search index
  internal/server/             local HTTP server with the Sync API
  theme/                       html/template files, CSS, JS — embedded with go:embed
  content/                     hand-written Markdown (business/, roles/, code/, home.md)
  data/                        curated JSON: module & group descriptions, A-to-Z stories, terms, pins
  dist/                        build output (git-ignored)
  docs/specs/                  this spec and later ones
  handbook.json                config: paths to the three app repos, site title, base URL
  Dockerfile  .gitlab-ci.yml  go.mod  README.md
```

### Configuration (`handbook.json`)

```json
{
  "siteTitle": "Carrybee Handbook",
  "baseURL": "/",
  "apps": [
    {"id": "hive",     "path": "../hive",     "label": "hive",     "audience": "ops"},
    {"id": "aladdin",  "path": "../aladdin",  "label": "aladdin",  "audience": "merchant"},
    {"id": "accounts", "path": "../accounts", "label": "accounts", "audience": "finance"}
  ]
}
```

Paths are relative to the config file; CI overrides them with `--apps-root`.

## 5. Components

Each package has one job and a small exported surface.

### 5.1 `internal/scan`

Input: an app id and its source path. Output: an `AppScan` struct, serialisable to JSON.

```go
type AppScan struct {
    App        string
    Commit     string             // short git hash of the app repo, "" if not a git repo
    ScannedAt  time.Time
    Modules    map[string]Module  // src/modules/<name> that exist as directories
    Routes     map[string]Route   // page groups derived from src/app
    APIDomains map[string][]string // API_ROUTES domain -> modules that reference it
    Statuses   []Status           // TransferStatus enum members and string status values
    Cycles     [][]string         // strongly connected groups of modules
}
type Module struct {
    Files int; Deps, UsedByModules, UsedByRoutes map[string]int; APIs, Shared map[string]int
}
```

Rules carried over from the prototype (and its bugs, fixed):

- Imports resolved from `@/` alias and relative paths; **commented-out lines are ignored**
  (the prototype counted two phantom modules from comments).
- Only directories that exist under `src/modules` count as modules.
- Route names: skip layout groups `(main) (protected) (dashboard_layout) (auth) (public)` and parallel
  slots `@x`; aladdin additionally drops `businesses/[businessId]` when followed by `[businessId]`.
- API domains: `API_ROUTES.<a>` and, for configured nested keys (aladdin `business`), `API_ROUTES.<a>.<b>`.
- Cycles: Tarjan's strongly connected components on the module→module graph.
- The three apps scan concurrently (one goroutine each, joined with `sync.WaitGroup`; no extra dependency).

Term linking (`scan.LinkTerms`) searches each module's source for each term's code names with word
boundaries, ignores names found in more than 12 modules, keeps the top 3 per app, then applies hand-checked
pins from `data/term-pins.json`.

### 5.2 `internal/content`

Loads `content/**/*.md` (front matter: `title`, `section`, `order`, `level`, `roles`, `summary`) and
`data/*.json`. Markdown is rendered with **goldmark** plus a small extension for:

- `{{term "cash-in-hand"}}` → dotted-underline term with popover
- `{{module "hive" "delivery-run"}}` → "where it lives" chip linking to the map
- `{{status 15}}` → coloured status pill
- `:::note`, `:::carrybee` ("If you already know courier"), `:::warning` callout blocks

Validation (`handbook check`) fails on: unknown term/module/status references, missing descriptions for
scanned modules, duplicate term ids, broken internal links.

### 5.3 `internal/site`

Renders every page with `html/template` into `dist/`:

- One base layout (top bar, sidebar, content, right rail) and page templates per kind
  (home, article, app map, terms, role, index).
- App map pages embed the interactive explorer (A-to-Z / Flow / Network) ported from the prototype as
  a standalone JS module reading `dist/data/<app>.json`.
- `dist/data/search.json`: one entry per term, module, page group, status, API domain and article.
- `dist/data/sync.json`: build time, app commits, and counts — read by the Sync pill.

### 5.4 `internal/server`

`handbook serve [--port 4321]`:

- Serves `dist/` with `net/http` `FileServer`.
- `GET /api/status` → `{ mode: "local", builtAt, commits }`.
- `POST /api/sync` → rescans, rebuilds into a temp dir, swaps it into `dist/` atomically, and returns a
  diff: modules/pages/API domains added or removed, terms newly linked, items needing descriptions.
  A mutex prevents concurrent syncs; a second request while one runs gets `409`.
- Binds to `127.0.0.1` only.

## 6. Sync button behaviour

The pill in the top bar reads `Synced 2 h ago · hive a1926599`.

- **Local** (`/api/status` answers): click → spinner → toast with the diff summary → page reloads.
  Errors (repo path missing, parse failure) show the message and keep the old site.
- **Deployed** (no API): click opens a small panel: last build time, commit per app, and
  "This site rebuilds automatically on every push." No button that does nothing.
- Newly scanned modules or terms without descriptions are listed as **Needs description**, linked to
  the file to edit.

## 7. UX and visual design

- **Brand tokens** from the apps' `globals.css`: primary bee yellow `oklch(0.865 0.177 90.382)` with
  near-black text on it; neutral greys (`--custom-*`); charcoal for dark surfaces; the brand's success,
  error and info colours. Light and dark themes plus a toggle (remembered in `localStorage`).
- **Layout**: top bar (mark, ⌘K search, Sync pill, theme toggle); left sidebar with the five sections and
  a yellow rail on the current page; content column ~72ch; maps and tables may go full width; right rail
  with "On this page" and related terms/modules on long pages. Phone: sidebar becomes a drawer.
- **Home**: short hero "Carrybee in 60 seconds" with a parcel strip (store → FMH → hub → LMH → buyer,
  ৳ flowing back); five role cards; two doors: "New to courier?" and "Already know logistics?".
- **Reading aids**: term popovers, ⓘ module popovers, level tags, "where it lives" chips,
  breadcrumbs, previous/next within a section.
- **Search (⌘K)**: grouped results by type, arrow-key navigation, Enter opens; works offline.
- **Diagrams** as inline SVG: status lifecycle state diagram, money waterfall, network tree.
- **Accessibility**: visible focus, `prefers-reduced-motion` respected, AA contrast in both themes,
  yellow only ever behind dark text.

## 8. Content to write (hand-written Markdown)

Migrated and expanded from the prototype guide and agent research:

- `home.md`, `business/courier-101.md`, `business/how-carrybee-works.md`, `business/statuses.md`
- `code/architecture.md`, `code/add-a-feature.md`, `code/conventions-and-gotchas.md`
- `roles/frontend.md`, `roles/backend.md`, `roles/qa.md`, `roles/product.md`, `roles/ops.md`
- `known-issues.md`: committed `AUTH_SECRET` in `hive/README.md`; hard-coded JWT in
  `hive/src/lib/constants.ts:15`; aladdin reject-urgent-approval uses the approve URL; accounts merchant and
  consignment reports share one endpoint; aladdin APP_ROUTES without pages; circular import groups.

Curated data migrated from the prototype into `data/`: module descriptions (104), group descriptions,
A-to-Z stories (3 apps), 185 terms, term pins.

## 9. Deployment

- `.gitlab-ci.yml`: `go test ./...` → `go build` → `handbook check` → `handbook build` → publish `dist/`
  to GitLab Pages. Clones the three app repos (read-only token from CI variables, never committed).
- `Dockerfile`: stage 1 builds the binary and `dist/`; stage 2 is `nginx:alpine` serving `dist/`.
  An alternative target runs `handbook serve` so Sync also works in a hosted container.

## 10. Error handling

- Missing app path: build continues with the other apps, marks that app "not scanned" in the UI, exits
  non-zero only under `check`.
- Unparseable file: logged with path, skipped; counted in the build summary.
- Content references to unknown items: warnings in `build`, errors in `check`.

## 11. Testing

- Table-driven unit tests for each scanner rule (import resolution, comment skipping, route naming,
  API domain extraction, cycle detection, term linking) using small fixture trees in `testdata/`.
- Golden test: scanning a fixture app produces an expected `AppScan` JSON.
- `handbook check` run against the real repos in CI.
- Site smoke test: build, then assert every internal link in `dist/` resolves and `search.json` contains
  every module and term.
- Manual: headless Chrome screenshots of home, a role page, an app map and terms, light and dark.

## 12. Delivery plan outline (detailed in the implementation plan)

1. Go skeleton, config, CLI; scanners with tests (parity with prototype numbers: hive 76 modules / 82 pages,
   aladdin 18 / 54, accounts 10 / 10).
2. Content loader, Markdown extensions, data migration, `check`.
3. Theme and site renderer: layout, home, articles, terms, search.
4. App map pages (port of the explorer).
5. Server and Sync.
6. Content writing (business, code, roles, known issues).
7. CI, Dockerfile, README; visual QA.

Independent tracks (scanner, theme, content writing) are suitable for parallel subagents.
