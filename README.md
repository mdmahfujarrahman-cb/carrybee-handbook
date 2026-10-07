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
| `make clean` | `rm -rf dist dist.old .dist-*` | Remove build output |
| `make help` | | List the targets |

Flags, for any of the three commands:

| Flag | Default | Meaning |
| --- | --- | --- |
| `-config` | `handbook.json` | Config file: site title and the list of apps with their paths |
| `-apps-root` | none | Folder that holds the app repos; overrides the paths in the config |
| `-out` | `dist` | Output folder (`build`, `serve`) |
| `-port` | `4321` | Port for `serve` (always bound to 127.0.0.1) |

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
| `docs` | Design spec and plans |

`dist/` is generated and not committed.

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

Run `make check` after editing. It fails if a term, module or status in a shortcode does not exist.

## Adding descriptions and terms

All in `data/`. The tool never edits these files, and Sync never touches them.

- **Module description** (`data/modules.json`): `"<app>": { "<module-id>": ["Title", "One or two plain sentences."] }`. The module id is the folder name under the app's `src/modules`.
- **Group description** (`data/groups.json`): same shape, for route groups, for example `"hive": { "g-mode": ["G-Mode (god mode)", "What it is for."] }`.
- **Term** (`data/terms.json`, `terms` array): `{"t": "Basket", "cat": "runs", "aka": ["basket_id"], "apps": ["hive"], "def": "A sealed bag of parcels."}`. `cat` is one of the `categories` ids, `aka` are the names used in code (they are used to link the term to modules), and the id is made from `t` unless you give `id`.
- **Term link pin** (`data/term-pins.json`): force a term to link to modules the code search misses. Key is the term's `t` name: `"Carry (carry amount)": { "hive": ["hub-payments", "hub-ledger"] }`.

## What Sync does

Clicking Sync rescans the three app repos, rebuilds the site into a temporary folder, swaps it in for `dist/`, and reports what changed: modules, pages and API calls added or removed, and new term-to-module links. If the build fails, the old `dist/` stays in place and the error is shown. Sync never writes to `content/` or `data/`.

## Modules that need a description

`make check` and `make build` end with lines like:

```text
needs description: module hive/hub-ledger has no description in data/modules.json
needs description: term some-term is not linked to any module (add a pin)
```

The first is a module found in code with no entry in `data/modules.json`. The second is a term that no module mentions in code. Both are listed, not errors. Add a description or a pin (see above) when you can.
