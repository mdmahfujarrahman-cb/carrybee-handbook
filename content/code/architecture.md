---
title: Architecture
section: code
order: 1
level: code
roles: frontend, backend, qa
summary: How the three Next.js apps are built and where each kind of code lives.
---
hive, aladdin and accounts are three separate repos built the same way. Once you can find your way around one, the other two look familiar. All three are frontends only: they call a shared Carrybee backend at `/api/v2`, which lives elsewhere. The best long-form references inside the repos are `aladdin/CLAUDE.md` and `hive/architecture.md`.

## Stack

| Layer | What is used |
|---|---|
| Framework | Next.js 16 App Router with React 19.2, TypeScript in strict mode, `output: 'standalone'` |
| Server state | TanStack Query v5, tables with TanStack Table v8 |
| Forms | react-hook-form with Zod schemas through `@hookform/resolvers/zod` |
| UI | shadcn/ui on Radix primitives, Tailwind CSS v4 |
| Auth | NextAuth v5 beta, JWT sessions, in `auth.ts` at the repo root |
| Real time | Centrifugo through the `centrifuge` client (hive and aladdin): live chat between sellers and riders ({{term "centrifugo"}}) |
| Push | OneSignal through `react-onesignal` (hive and aladdin) ({{term "onesignal-push"}}) |
| Package manager | pnpm |

Next 16 is newer than most tutorials and some AI tools. `accounts/AGENTS.md` says it directly: read the guide in `node_modules/next/dist/docs/` before relying on what you remember. One visible change: the request interceptor is `src/proxy.ts`, which replaced `middleware.ts`.

## Folder layout

```text
src/
├── app/          App Router pages, layouts and route handlers
├── modules/      one folder per feature (see below)
├── components/   shared UI: ui/ (shadcn), form/, table/, layout/
├── lib/          api-client/, routes/, permission/, constants.ts, utils
├── hooks/        shared hooks
├── contexts/     React providers (Query, Theme, loading overlay)
├── types/
└── proxy.ts      runs before every page request
```

## Route groups and parallel routes

Folder names in brackets are route groups. They organise pages and share a layout without changing the URL.

| App | Main groups |
|---|---|
| hive | `(auth)` for login and logout, `(main)/(protected)/(dashboard_layout)/...` for everything behind the sidebar, with feature groups such as `(hub)` and `(merchant)` inside |
| aladdin | `(public)`, `(protected)` (holding `(business)` and `(merchant)`), `(developers-api)` for the webhook docs |
| accounts | `(auth)` for login, `(main)` for the panel |

aladdin uses two more App Router features. Its dashboard at `businesses/[businessId]/dashboard` is built from **parallel routes**: folders such as `@orders` and `@payments` are slots that `dashboard/layout.tsx` receives as props and lays out, each loading on its own. It also uses **intercepted routes**: `@modal/(.)profile` and `@modal/(.)request-pickup` open as modals during client navigation, and a hard refresh renders the real page instead. Read the "Routing" section of `aladdin/CLAUDE.md` before you add a sibling route there.

## Module anatomy

Feature code lives in `src/modules/<feature>`. Every module on the [hive](../apps/hive.html), [aladdin](../apps/aladdin.html) and [accounts](../apps/accounts.html) maps is one of these folders.

| Folder | Holds |
|---|---|
| `components/` | The module's UI |
| `hooks/` | TanStack Query hooks. hive splits them into `hooks/api/list`, `detail` and `mutation` |
| `services/` | `api.ts` makes raw HTTP calls; `service.ts` converts snake_case to camelCase and returns a `ServiceResponse<T>`; `mapper.ts` holds larger transformations |
| `types/` | API response types in snake_case plus the camelCase types derived from them |
| `validations/` | Zod schemas for forms |
| `utils/` | Constants, helpers and, in hive, `query-keys.ts` |

A typical read goes: component, query hook, `service.ts`, `api.ts`, `restApiClient` in `src/lib/api-client`, backend, then back up with `mapSnakeToCamel` applied. The diagram at the end of `hive/architecture.md` walks through it.

## The two route files

Each app has `src/lib/routes/app-routes.ts` and `src/lib/routes/api-routes.ts`. Treat them as the index of the app.

- `APP_ROUTES` in `app-routes.ts` builds every frontend URL, for example `APP_ROUTES.business.orders.details(businessId, orderId)` in aladdin. Don't write URLs as strings in components.
- `API_ROUTES` in `api-routes.ts` lists every backend endpoint the app calls, grouped by domain. The [API domain index](api-index.html) is generated from these files.

## Auth, proxy and permissions

Login differs per app (see the table below), but the shape is shared. `auth.ts` configures NextAuth; after login the session holds the user's permission keys in a compressed form.

`src/proxy.ts` runs on every page request. It sends logged-out users to the login page, looks up which permissions the requested path needs, and redirects users who lack them. The path-to-permission map is `route-permissions.ts` (in accounts, `src/lib/route-permission.ts`).

Inside a page, UI is gated by permission components: `PermissionGate` on the server and `client-permission-gate.tsx` or the `usePermissions()` hook on the client, all in `src/lib/permission/`. hive also has `PermissionWrapper` in `src/components/permission/`. hive's sidebar is built in `src/hooks/use-menu.ts`, and each menu item is shown only if `validateRouteAccess` passes for its URL.

## Data fetching and forms

Server components `await params` and `searchParams` (both are Promises in Next 16) and call module services. A common pattern starts the service call on the server, passes the promise to a client component, and unwraps it there with React's `use()` inside `<Suspense>`.

Client components use TanStack Query. Query keys come from a key factory, so a mutation can invalidate exactly the lists it changed. In aladdin the factory is `src/hooks/api/query-keys.ts`; in hive each module has its own `utils/query-keys.ts`.

Forms take a Zod schema from the module's `validations/`, connect it to react-hook-form, render field components from `src/components/form/`, and submit through the service. Backend field errors can be mapped back onto the form.

## How the apps differ

| | hive | aladdin | accounts |
|---|---|---|---|
| Users | Carrybee staff | Sellers | Finance and legal |
| Size | 76 modules, 82 pages | 18 modules, 54 pages | 10 modules, 9 pages |
| Login | Backend-brokered {{term "sso"}}, then a hub is selected; impersonation | Phone and password, OTP at signup; impersonation | Phone and OTP |
| Permissions | Many keys (`src/lib/permission/permissions.ts`) | Per-business keys, URL must match the selected business | Two keys: `invoice_and_billing.revenue_recognition.view` and `invoice_and_billing.legal_panel.view` |
| Navigation | Sidebar from `src/hooks/use-menu.ts` | Business routes under `/businesses/:businessId/...` | Sidebar in `src/components/layout/app-sidebar.tsx` |
| Languages | Mostly English; small `en.json` / `bn.json` used by a few files | English and Bangla throughout (`src/lib/translation/locales/en.json`, `bn.json`) | English |
| Zod | 3 | 3 | 4 |
| Real time and push | Centrifugo, OneSignal | Centrifugo, OneSignal | None |
| Module boundaries lint | `boundaries/element-types` on, but modules may import any other module | `boundaries/element-types` commented out | No boundaries plugin |
| Pre-commit | lint-staged, then a full build | lint-staged, then a full build | No hooks |
| Commit messages | commitlint, Conventional Commits | commitlint, Conventional Commits | Not enforced |

Module and page counts come from the scan behind the app maps and update on every build.
