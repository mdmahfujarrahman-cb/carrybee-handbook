---
title: Conventions and gotchas
section: code
order: 3
level: code
roles: frontend, backend, qa
summary: The house rules for the three repos, and the traps that catch new people.
---
## Conventions

**pnpm only.** Each repo has a `pnpm-lock.yaml`. Don't run `npm install` or `yarn`; a second lock file causes conflicts.

**Conventional Commits.** hive and aladdin enforce `type(scope): subject` with commitlint, for example `fix(services): handle token expiry on 401`. aladdin's allowed scopes are listed in `aladdin/CLAUDE.md`. accounts has no commitlint, but use the same format there.

**The pre-commit hook builds the app.** In hive and aladdin, `.husky/pre-commit` runs lint-staged (Prettier and ESLint on staged files) and then a full `pnpm build`. Expect a commit to take a few minutes. If the build fails, the commit is blocked. accounts has no hooks.

**Lint with zero warnings in aladdin.** aladdin runs ESLint with `--max-warnings=0`, so a warning fails the lint. Its custom `tailwind-colors/no-hardcoded-colors` rule warns on hex colours, so use Tailwind colour classes. `no-explicit-any` and `no-unused-vars` are switched off there, so remove unused code and avoid `any` yourself.

**Prettier.** Width 120, single quotes, trailing commas, two-space indent, Tailwind class sorting (`prettier.config.js` in hive and aladdin).

**SonarQube rules.** Write code that would pass the usual SonarQube checks: keep functions short and shallow, no nested ternaries, no commented-out code, no unused imports or variables, no hard-coded secrets, no empty `catch`, `unknown` instead of `any`. No Sonar configuration was found in the three repos or their CI files, so you can't run it locally; ESLint, `tsc` and the build are the checks you can run.

**URLs come from the route files.** Build page links with `APP_ROUTES` and endpoints with `API_ROUTES`. A string URL in a component breaks the next time the route moves, and it hides the page from the app maps.

**Case conversion happens in services.** API types are snake_case, everything above the service layer is camelCase. Convert in `service.ts` with `mapSnakeToCamel` and `mapCamelToSnake`, not in components.

## Gotchas

**Module isolation is not enforced.** aladdin configures `eslint-plugin-boundaries` but the `boundaries/element-types` rule is commented out in `aladdin/eslint.config.mjs`. hive has the rule on, but it allows any module to import any other module. accounts does not use the plugin. Cross-module imports are a code review matter in all three apps.

**There are circular imports.** Some modules import each other in a loop, which makes them hard to change or move on their own. The app maps flag them:

| App | Module groups that import each other in a loop |
| --- | --- |
| hive | `cluster`, `region`, `users`, `warehouses` |
| hive | `basket`, `no-entry-orders`, `orders` |
| hive | `petty-cash`, `reconciliation` |
| hive | `receivable-postpaid-invoices`, `receivable-prepaid-invoices` |
| aladdin | `businesses`, `customers`, `dashboard`, `invoices`, `orders`, `payments`, `stores` |
| accounts | none |

Don't add to a loop. If your change needs something from a module in one of these groups, check whether it belongs in `src/lib` or `src/components` instead.

**No tests.** None of the three apps has a test runner or a single test file. Type check, lint, build and a manual run are the only safety net. QA covers the gap; see the [QA page](../roles/qa.html).

**Next 16 is not the Next.js you remember.** `params` and `searchParams` are Promises, `middleware.ts` is now `src/proxy.ts`, and Turbopack is the default dev server. `accounts/AGENTS.md` asks you to read the docs shipped in `node_modules/next/dist/docs/` before writing code, and that advice applies to all three apps.

**Old status numbers live on in comments.** `hive/src/lib/constants.ts` has commented-out lines below the `TransferStatus` enum with an older numbering (for example `// Found = 40` and `// OnTheWayToSubSort = 42`). Today 40 means a return on the way to sub-sort and 42 means a reverse pickup is waiting. Trust the live enum values, never the comments. See [Statuses](../business/statuses.html).

**Two names for the same thing.** Hubs are called `warehouses` in many hive paths and menus. Riders are always `agent` in code. A parcel is an order, a consignment or a parcel depending on the screen. The [courier terms](../terms.html) list the aliases.

**aladdin's intercepted modals need real pages.** Every `@modal/(.)x` route needs a matching `[businessId]/x/page.tsx`, or a hard refresh on that URL fails. `update-password` has only the modal today.

**Not every pattern in the code is a good one.** Before you copy a pattern, ask a reviewer whether it is still the way to do it. Some, such as a hard-coded token, must not spread.
