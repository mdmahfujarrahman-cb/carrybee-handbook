---
title: Add a feature
section: code
order: 2
level: code
roles: frontend
summary: Step by step, from a new route to a checked build, in any of the three apps.
---
This is the path for a typical feature: a new page that lists something from the backend and has a form to create it. The steps are the same in hive, aladdin and accounts; differences are called out. Read [Architecture](architecture.html) first if the folder names are new to you. `hive/architecture.md` ends with a "New Module Checklist" that matches these steps.

## 1. Add the route

Add the URL to `APP_ROUTES` in `src/lib/routes/app-routes.ts`. Use a function when the URL has parameters:

```ts
reports: {
  index: (businessId: string) => `/businesses/${businessId}/reports`,
},
```

In aladdin, business pages live under `/businesses/:businessId/...` and take the business ID as the first argument.

## 2. Create the page

Create `page.tsx` under the right route group in `src/app`, for example `(protected)/(business)/businesses/[businessId]/reports/page.tsx` in aladdin or a folder inside `(main)/(protected)/(dashboard_layout)` in hive. The folder path must produce the URL you added in step 1, since route groups do not appear in the URL.

In Next 16, `params` and `searchParams` are Promises. `await` them in the server component, start the service call, and pass the promise or data to client components inside `<Suspense>`.

If you add a page next to a segment that has parallel-route slots (aladdin's dashboard, or `[businessId]` with its `@modal` slot), check that each slot has a `default.tsx`, or a hard refresh will 404.

## 3. Create the module folder

Create `src/modules/<feature>/` with the folders you need:

```text
src/modules/reports/
├── components/
├── hooks/
├── services/   api.ts, service.ts
├── types/      index.ts
├── utils/      constants, query-keys.ts (hive)
└── validations/
```

Keep the module's imports to its own files and the shared folders (`src/components`, `src/lib`, `src/hooks`, `src/types`). Import from another module only when you must, and never in a way that creates a loop. Nothing enforces this in aladdin or accounts; see [Conventions and gotchas](conventions-and-gotchas.html).

## 4. Add the API call

1. Add the endpoint to `API_ROUTES` in `src/lib/routes/api-routes.ts`, inside the domain it belongs to.
2. In `services/api.ts`, call it through `restApiClient` (`get`, `post`, `put`, `patch`, `del`). The client attaches the session token and signs the user out on a 401.
3. In `services/service.ts`, convert the response with `extractBackendData` and `mapSnakeToCamel`, and return `{ data, message, error, pagination }`.
4. Declare the response type in `types/index.ts` in snake_case, and derive the camelCase type from it.

The backend always replies with `{ data, error, message, pagination?, causes? }`. `causes` holds field-level validation errors that you can map back onto a form.

## 5. Add query keys and hooks

Add keys for the new data to the key factory: `src/hooks/api/query-keys.ts` in aladdin, `utils/query-keys.ts` inside the module in hive. Then write a `useQuery` hook for the list and a `useMutation` hook for the create. In the mutation, invalidate the list key when it settles so the list refreshes.

## 6. Build the form

Write a Zod schema in `validations/`, connect it with `zodResolver` to `useForm`, and use the field components in `src/components/form/`. Export the type inferred from the schema rather than writing it twice.

## 7. Gate it by permission

1. Add the route's required permission to `route-permissions.ts` in `src/lib/permission/` (in accounts, `src/lib/route-permission.ts`). `src/proxy.ts` uses this map to redirect users who lack access.
2. If the permission key is new, add it to `permissions.ts`. The key itself is created on the backend; ask the backend team for its exact name.
3. Wrap buttons and sections that need extra rights in `PermissionGate` (server) or the client gate or `usePermissions()` hook.
4. In hive, add a menu entry in `src/hooks/use-menu.ts` with `show: validateRouteAccess(compressedPermissions, APP_ROUTES.yourRoute)`.

## 8. Add translations (aladdin)

aladdin shows every string in English and Bangla. Add each key to both `src/lib/translation/locales/en.json` and `bn.json` and type it in `locales/types.ts`. Use `useTranslation()` in client components and `getServerTranslations()` in server components. Run `pnpm i18n:test` to catch missing or duplicate keys, and `pnpm i18n:export` when translators need the spreadsheet.

## 9. Check before you commit

There are no automated tests in any app, so these checks are what you have:

```bash
pnpm exec tsc --noEmit   # type check (no script exists for it)
pnpm lint                # ESLint; aladdin fails on any warning
pnpm build               # production build
```

In hive and aladdin the pre-commit hook runs lint-staged and then a full `pnpm build`, so a commit takes a while and a broken build blocks it. Then open the page in `pnpm dev`, log in as a user with and without the permission, and check both.
