# Carrybee Handbook

An onboarding handbook for new Carrybee employees. Carrybee Express is a parcel courier in Bangladesh: sellers book parcels, hubs sort and move them, riders deliver them and collect cash, and the money is settled back to sellers. Three Next.js apps run that business. **aladdin** is the seller panel, **hive** is the staff panel for hubs, riders, sorting, returns, cash and payouts, and **accounts** is the finance and legal panel. This tool scans those three codebases (modules, routes, API calls, order statuses), combines the result with hand-written pages, and builds a static, branded site in `dist/`: how the business works, a guide per role, a map of each app's modules, an API index, a glossary of courier terms and a network view of module dependencies.

## Requirements

- Go 1.24 or newer.
- The three app repos checked out as sibling folders of this one: `../hive`, `../aladdin`, `../accounts`. If they live elsewhere, pass `--apps-root <dir>` (a folder that contains the three repos).

## Quick start

```bash
make run
```

Open <http://127.0.0.1:4321>. After you pull new code in the app repos, click **Sync** in the page header. The handbook rescans, rebuilds and shows what changed.

## Commands

| Make target | Equivalent | What it does |
| --- | --- | --- |
| `make run` | `go run ./cmd/handbook serve` | Serve the site and the Sync button on 127.0.0.1:4321 |
| `make build` | `go run ./cmd/handbook build` | Scan and write the static site to `dist/` |
| `make check` | `go run ./cmd/handbook check` | Validate references in `content/`; list modules without a description |
| `make test` | `go vet ./... && go test -count=1 ./...` | Vet and run the tests |
| `make fmt` | `gofmt -w .` | Format Go code |
| `make share` | `./scripts/share.sh` | Build `share/carrybee-handbook.zip` for people without the repos (see below) |
| `make clean` | `rm -rf dist dist.old .dist-*` | Remove build output |
| `make help` | | List the targets |

Flags, for any of the three commands:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-config` | `handbook.json` | Config file: site title and the list of apps with their paths |
| `-apps-root` | none | Folder that holds the app repos; overrides the paths in the config |
| `-out` | `dist` | Output folder (`build`, `serve`) |
| `-port` | `4321` | Port for `serve` (always bound to 127.0.0.1) |
| `-allow-snapshot` | off | `check` only: pass even when apps were loaded from `snapshot/scan.json` instead of scanned live |

Example: `go run ./cmd/handbook build --apps-root ~/code/carrybee --out public`.

You can also open `dist/index.html` straight from disk. Everything works offline except the network view, which loads d3 from a CDN.

## Folder map

| Path | Contents |
| --- | --- |
| `cmd/handbook` | The CLI (`build`, `serve`, `check`) |
| `internal/config` | Reads `handbook.json` |
| `internal/scan` | Reads the app repos: modules, routes, API calls, status enum, import cycles |
| `internal/content` | Markdown pages, shortcodes, callouts, data files, reference checks |
| `internal/site` | Builds the site: pages, generated pages, search index, term linking |
| `internal/server` | The `serve` command: static files, `/api/status`, Sync |
| `theme` | HTML templates, CSS, JS and the logo |
| `content` | Hand-written pages (Markdown) |
| `data` | Hand-written JSON: module and group descriptions, terms, pins |
| `snapshot` | `scan.json`: the structure found by the last complete scan (committed) |
| `scripts` | `share.sh` and the start scripts packed into the shared ZIP |
| `docs` | Design spec and plans |

`dist/` and `share/` are generated and not committed.

## Editing content

Pages live in `content/**/*.md`. Each starts with front matter:

```markdown
---
title: Courier 101
section: business
order: 1
level: basics
roles: frontend, qa
summary: One line shown in cards and search.
---
```

`title`, `section` (folder name), `order` (position inside the section), `level` (`basics`, `carrybee` or `code`), `roles` (comma list: frontend, backend, qa, product, ops) and `summary`.

Shortcodes inside the Markdown:

| Shortcode | Renders |
| --- | --- |
| `{{term "cod"}}` | A glossary term, with a hover definition, using the term's name |
| `{{term "basket" "basket"}}` | The same term, with your own text. Use this mid-sentence so the term's capital letter does not show |
| `{{module "hive" "delivery-run"}}` | A chip linking to that module on its app page |
| `{{status 15}}` | A status pill such as "15 Delivered" |
| `{{diagram "statuses"}}` | An inline SVG diagram from `theme/diagrams/<name>.html` (`statuses`, `money`, `network`). Put it on its own line |

Callouts:

```text
:::note
Plain aside.
:::

:::carrybee
How Carrybee uses a word differently from the outside world.
:::

:::warning
Something that bites.
:::
```

Link to other pages with relative `.html` links, for example `[Courier 101](courier-101.html)` or `[Statuses](../business/statuses.html)`. Use spaced pipes in tables (`| a | b |`, `| --- | --- |`).

Run `make check` after editing. It fails if a term, module, status or diagram in a shortcode does not exist, or if a link points at a missing page or a missing `#heading` id. Every heading gets an id from its text (`## Money flow` becomes `#money-flow`).

## Adding descriptions and terms

All in `data/`. The tool never edits these files, and Sync never touches them.

- **Module description** (`data/modules.json`): `"<app>": { "<module-id>": ["Title", "One or two plain sentences."] }`. The module id is the folder name under the app's `src/modules`.
- **Group description** (`data/groups.json`): same shape, for route groups, for example `"hive": { "g-mode": ["G-Mode (god mode)", "What it is for."] }`.
- **Term** (`data/terms.json`, `terms` array): `{"t": "Basket", "cat": "runs", "aka": ["basket_id"], "apps": ["hive"], "def": "A sealed bag of parcels."}`. `cat` is one of the `categories` ids, `aka` are the names used in code (they are used to link the term to modules), and the id is made from `t` unless you give `id`.
- **Term link pin** (`data/term-pins.json`): force a term to link to modules the code search misses. Key is the term's `t` name: `"Carry (carry amount)": { "hive": ["hub-payments", "hub-ledger"] }`.

Term ids must be unique; the build stops and names the duplicate.

## What Sync does

Clicking Sync rescans the three app repos, rebuilds the site into a temporary folder, swaps it in for `dist/`, and reports what changed: modules, pages and API calls added or removed, and new term-to-module links. The results stay in a panel under the Sync button, with warnings and every item that needs a description grouped by the `data/` file to edit; click **Reload to see changes** to load the new pages. If the build fails, the old `dist/` stays in place and the error is shown. Sync never writes to `content/` or `data/`.

The Sync button also shows whether the handbook is current: **Up to date** or **Update available**, with the time of the last sync. It checks each app repo for commits after the built commit and for `.ts`/`.tsx` files under `src/` changed since the last build (checked at most every 30 seconds; the page asks every minute and when you come back to the tab). Hover the button for the per-app details. Sync only accepts requests from the handbook's own page.

## Sharing the handbook

To give the handbook to someone who has none of the app repos (and maybe no Go), run:

```bash
make share
```

It needs the three repos next to this folder. It scans them (refreshing `snapshot/scan.json`), runs the tests, builds the handbook program for macOS (Apple silicon and Intel), Linux and Windows, prebuilds the site from the snapshot and writes `share/carrybee-handbook.zip`. Send that file.

The recipient unzips it and runs `./start.sh` (Mac, Linux) or double-clicks `start.bat` (Windows); the handbook opens on <http://127.0.0.1:4321>. They can also just open `dist/index.html`. `README-FIRST.txt` in the ZIP says the same. `HANDBOOK_PORT` changes the port and `HANDBOOK_NO_BROWSER=1` stops the start scripts opening a browser.

They see the whole handbook: app maps, search, terms and statuses. Sync is off for them: the pill is grey and reads **Snapshot · built 7 Oct**, its tooltip says why, and clicking it shows each app's commit and scan date. To give them newer content, pull the repos and run `make share` again.

### The scan snapshot

Every build that scans all apps cleanly writes `snapshot/scan.json`: modules, routes, API domains, statuses and term links. It holds names and counts only, never source code. Commit it, so a plain `git clone` without the app repos still builds a complete handbook. When an app's folder is missing, the build takes that app from the snapshot instead of printing "not scanned", and the Sync button is disabled (`/api/status` reports `canSync: false` and `POST /api/sync` returns 409). `make check` still fails for snapshot apps, as CI should scan live code; pass `-allow-snapshot` (`go run ./cmd/handbook check -allow-snapshot`) to accept them.

## Modules that need a description

`make check` and `make build` end with lines like:

```text
needs description: module hive/hub-ledger has no description in data/modules.json
needs description: page group hive/oms has no description in data/groups.json
needs description: term some-term is not linked to any module (add a pin in data/term-pins.json)
```

Each line names the file to edit: a module or a route group found in code with no description, or a term that no module mentions in code. `make build` only lists them; `make check` fails until they are fixed. Add a description or a pin (see above).
