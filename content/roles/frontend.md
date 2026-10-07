---
title: Frontend developer
section: roles
order: 1
level: code
roles: frontend
summary: App architecture, module anatomy, conventions and where to start.
---
You will work in one or more of three Next.js 16 apps that share a layout and a style. Learn the business first, because almost every screen is a step in a parcel's journey or its money trail. Then learn one app's structure well; the other two follow the same pattern.

## What you care about

- **Where code lives.** Pages in `src/app`, feature code in `src/modules/<feature>`, every URL in `app-routes.ts`, every backend call in `api-routes.ts`.
- **Who can see what.** Permissions decide which pages, menu items and buttons a user sees. Most "it doesn't show up" bugs are permission bugs.
- **Statuses.** Lists, filters, tabs and buttons depend on an order's status number. You will meet `TransferStatus` everywhere.
- **Safe changes without tests.** There are no automated tests. Type check, lint and build are your checks, plus a manual run as the right user.
- **The traps.** Circular imports, unenforced module boundaries and Next 16 changes.

## Read in this order

1. [How Carrybee works](../business/how-carrybee-works.html): the business, the three apps and the A-to-Z journey.
2. [Architecture](../code/architecture.html): stack, folders, route groups, auth and the per-app differences.
3. [Add a feature](../code/add-a-feature.html): the steps from route to checked build.
4. [Conventions and gotchas](../code/conventions-and-gotchas.html): house rules and traps.
5. The map of the app you will work on: [aladdin](../apps/aladdin.html), [hive](../apps/hive.html) or [accounts](../apps/accounts.html). Start with `orders` in aladdin and `delivery-run` in hive.
6. [Statuses](../business/statuses.html): numbers, text values and the merchant tabs.
7. [API domain index](../code/api-index.html): which module calls which backend domain.
8. [Known issues](../known-issues.html): bugs you should not copy.

## Cheat sheet

### Commands

| Command | What it does |
|---|---|
| `pnpm dev` | Dev server (Turbopack) |
| `pnpm exec tsc --noEmit` | Type check; there is no script for it |
| `pnpm lint` | ESLint; aladdin fails on any warning |
| `pnpm build` | Production build; also run by the pre-commit hook in hive and aladdin |
| `pnpm format:fix` | Prettier write (hive, aladdin) |
| `pnpm i18n:test` | Check English and Bangla keys (aladdin) |

### Module anatomy

```text
src/modules/<feature>/
├── components/    UI
├── hooks/         TanStack Query hooks (hive: hooks/api/list, detail, mutation)
├── services/      api.ts (raw calls), service.ts (snake to camel), mapper.ts
├── types/         snake_case API types and camelCase types derived from them
├── validations/   Zod schemas
└── utils/         constants, query-keys.ts (hive)
```

### Where API calls live

| What | Where |
|---|---|
| Endpoint paths | `src/lib/routes/api-routes.ts` (`API_ROUTES`) |
| HTTP client | `src/lib/api-client/index.ts` (`restApiClient`) |
| Raw calls | `src/modules/<feature>/services/api.ts` |
| Conversion and error handling | `src/modules/<feature>/services/service.ts` |
| Query keys | aladdin `src/hooks/api/query-keys.ts`; hive `src/modules/<feature>/utils/query-keys.ts` |
| Backend base URL | `NEXT_PUBLIC_API_BASE_URL`, prefix `/api`, version `/v2` |

### How permissions gate the UI

| Layer | Where | What it does |
|---|---|---|
| Route | `src/proxy.ts` with `route-permissions.ts` | Redirects users who lack the path's permission |
| Menu (hive) | `src/hooks/use-menu.ts` | Shows a menu item only if `validateRouteAccess` passes |
| Server component | `PermissionGate` in `src/lib/permission/` | Renders children only with the given permissions |
| Client component | `client-permission-gate.tsx`, `usePermissions()`; hive also `PermissionWrapper` | Same check on the client |

Permissions travel compressed in the NextAuth session. accounts has only two keys; hive and aladdin have many. To test a gate, log in as a user without the key and check the page redirects and the button disappears.
