---
title: Known issues
section: reference
order: 1
level: code
roles: frontend, backend, qa
summary: Security findings and suspected bugs, each with file and line.
---
These problems were found while reading the three codebases for this handbook. Each entry says what the problem is, where it is, why it matters and a suggested fix. Line numbers are from the commits scanned for the current build and can drift; search for the quoted name if a line has moved. Secret values are deliberately not reproduced here.

:::warning
Findings listed here are real problems in the code today. Do not copy the patterns they describe.
:::

| # | Issue | Where | Kind |
|---|---|---|---|
| 1 | Committed auth secret | `hive/README.md:12` | Security |
| 2 | Hard-coded access token | `hive/src/lib/constants.ts:15` | Security |
| 3 | Reject urgent approval calls the approve URL | `aladdin/src/lib/routes/api-routes.ts:77` | Suspected bug |
| 4 | Consignment and merchant reports share an endpoint | `accounts/src/lib/routes/api-routes.ts:43-44` | Suspected bug |
| 5 | Routes defined with no page | `aladdin/src/lib/routes/app-routes.ts` | Dead links |
| 6 | Circular imports between modules | hive (4 groups), aladdin (1 group) | Maintainability |
| 7 | Module boundaries rule switched off | `aladdin/eslint.config.mjs:89` | Maintainability |
| 8 | No automated tests | all three apps | Quality |

## 1. Committed AUTH_SECRET in hive's README

**What.** `hive/README.md` line 12 contains an `AUTH_SECRET=` line with a real-looking value, as part of the setup instructions.

**Impact.** NextAuth signs and encrypts session tokens with this secret. Anyone with read access to the repo, or to any copy or fork of it, can forge or decrypt sessions for any environment that uses the same value.

**Fix.** Replace the value with a placeholder such as `AUTH_SECRET=<generate with: npx auth secret>`. Rotate the secret in every environment that used it, since it stays in git history.

## 2. Hard-coded ACCESS_TOKEN in hive's constants

**What.** `hive/src/lib/constants.ts` line 15 exports `ACCESS_TOKEN`, a full JWT written into the source.

**Impact.** Nothing in `hive/src` imports it today, so it is not used at runtime, but it sits in the repo and its history. If it is still valid, it gives whoever holds it the access of the user it was issued to. Even if expired, it is a pattern that gets copied.

**Fix.** Delete the constant; tokens belong in the session. Ask the backend team to revoke the token.

## 3. aladdin's reject urgent approval uses the approve URL

**What.** In `aladdin/src/lib/routes/api-routes.ts`, line 76 defines `approveUrgentApprovalOrder` and line 77 defines `rejectUrgentApprovalOrder`. Both return `/businesses/${businessId}/urgent-approvals/approve`. The reject call in `aladdin/src/modules/high-alert/services/api.ts` posts to it.

**Impact.** When a seller rejects a rider's request (return, price change, exchange, partial delivery, paid return) the request goes to the approve endpoint. The payload does carry `action: 'reject'` and a `rejectReason` (`aladdin/src/modules/high-alert/types/index.ts`), so the backend may handle it correctly if it reads `action`. Whether it does is not confirmed in code. If it doesn't, a seller's rejection becomes an approval, which changes the parcel's outcome and money.

**Fix.** Confirm with the backend how `/urgent-approvals/approve` treats `action: 'reject'`. If a separate reject endpoint exists, point `rejectUrgentApprovalOrder` at it. Either way, QA should test reject end to end.

## 4. accounts' consignment and merchant reports share one endpoint

**What.** In `accounts/src/lib/routes/api-routes.ts`, `reports.consignmentsReports` (line 43) and `reports.merchantReports` (line 44) are both `/reporting/csv-by-consignments`. hive has the same pair in `hive/src/lib/routes/api-routes.ts:671-673`, where the merchant line carries the comment `// change after backend_implementation`.

**Impact.** The merchant report page in accounts downloads a consignment report. Finance may believe they have a per-merchant export when they don't.

**Fix.** Once the backend has a merchant report endpoint, update both apps. Until then, hide or label the merchant report page.

## 5. aladdin APP_ROUTES with no page

**What.** `APP_ROUTES` in `aladdin/src/lib/routes/app-routes.ts` builds URLs for pages that do not exist under `src/app`. Examples checked against the page files:

| Route key | URL | Line |
|---|---|---|
| `public.about`, `public.contact`, `public.help`, `public.pricing` | `/about`, `/contact`, `/help`, `/pricing` | 7-12 |
| `authentication.verifyEmail` | `/verify-email` | 22 |
| `merchant.settings`, `merchant.account`, `merchant.billing` | `/settings`, `/account`, `/billing` | 31-34 |
| `business.index` | `/businesses` | 64 |
| `business.updatePassowrd` | `/businesses/:id/update-password` (modal only; 404 on hard refresh) | 74 |
| `business.settings` | `/businesses/:id/settings` | 77 |
| `business.invoices.create` | `/businesses/:id/invoices/create` | 116 |
| `business.analytics.sales` and siblings | `/businesses/:id/analytics/sales` | 123-126 |
| `business.payments.index` | `/businesses/:id/payments` (the real page is `/payment`) | 130 |

**Impact.** Any link or redirect built from these keys lands on a 404. They also make the route file a less reliable index of the app.

**Fix.** Delete the keys nobody uses, or add the pages. For `update-password`, add a real `[businessId]/update-password/page.tsx` next to the intercepted modal.

## 6. Circular imports

**What.** Groups of modules import each other in a loop. The scanner finds 4 groups in hive and 1 in aladdin; the app maps show them.

| App | Group |
|---|---|
| hive | `cluster`, `region`, `users`, `warehouses` |
| hive | `basket`, `no-entry-orders`, `orders` |
| hive | `petty-cash`, `reconciliation` |
| hive | `receivable-postpaid-invoices`, `receivable-prepaid-invoices` |
| aladdin | `businesses`, `customers`, `dashboard`, `invoices`, `orders`, `payments`, `stores` |

**Impact.** A module in a loop cannot be changed, tested or moved without touching the others. Loops can also cause `undefined` imports at load time when modules initialise in the wrong order.

**Fix.** Move the shared pieces (usually types, constants or small helpers) into `src/types`, `src/lib` or `src/components`, then switch on a lint rule that blocks new loops, for example `import/no-cycle`.

## 7. Module boundaries rule switched off

**What.** aladdin configures `eslint-plugin-boundaries` but the rule that enforces it, `boundaries/element-types`, is commented out from line 89 of `aladdin/eslint.config.mjs`. Only `no-unknown` and `no-unknown-files` run. hive has the rule on but allows any module to import any other (`hive/eslint.config.mjs:54-66`). accounts does not use the plugin.

**Impact.** Nothing stops a module from reaching into another module's internals, which is how the loops in issue 6 formed.

**Fix.** Turn the rule back on as a warning first, fix the reported imports, then raise it to an error.

## 8. No automated tests

**What.** None of the three apps has a test runner or any `*.test.*` or `*.spec.*` file under `src`.

**Impact.** Every change relies on type checks, lint, the build and manual testing. Money calculations, such as the hub deposit formula, and status-driven logic have no regression safety net.

**Fix.** Start small: add Vitest and unit-test pure functions first (formulas, mappers, permission helpers), then add a few end-to-end journeys from the [QA page](roles/qa.html).
