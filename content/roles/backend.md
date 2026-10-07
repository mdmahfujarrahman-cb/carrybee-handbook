---
title: Backend developer
section: roles
order: 2
level: code
roles: backend
summary: Which app calls which API, status enums and money formulas.
---
The backend is not in these repos, but the three frontends are its only web clients. They tell you which endpoints are in use, what shape the responses must have, which status numbers mean what, and how money is calculated on screen. This page collects that view.

## What you care about

- **Which frontend calls which endpoint.** Before you change or remove an endpoint, find every caller.
- **The response envelope.** Every frontend expects `{ data, error, message, pagination?, causes? }`.
- **Status numbers.** The frontends hard-code the meaning of each `TransferStatus` number.
- **Money fields.** Which amounts the frontend reads, which it computes, and where paisa values exist.
- **Permission keys.** Pages and menus are shown by permission key, so a renamed key hides a page.

## Read in this order

1. [How Carrybee works](../business/how-carrybee-works.html): the journey and the money flow.
2. [API domain index](../code/api-index.html): every API domain, the modules that call it and the pages that load them.
3. [Statuses](../business/statuses.html): every status number and the merchant tabs.
4. [Architecture](../code/architecture.html): how the frontends call the API and check permissions.
5. [Courier terms](../terms.html): the field names behind each term (for example `collectable_amount`, `cod_fee`).
6. [Known issues](../known-issues.html): two of them are endpoint mix-ups you can confirm or fix from your side.

## Cheat sheet

### Who calls what

All three apps call the same backend at `NEXT_PUBLIC_API_BASE_URL` + `/api` + `/v2`. Endpoint paths are listed in each app's `src/lib/routes/api-routes.ts`, grouped by domain.

| App | Domains in `api-routes.ts` | Notes |
|---|---|---|
| [hive](../code/api-index.html#api.hive) | About 70, for example `orders`, `run`, `basket`, `centralSorting`, `ptn`, `hubPayment`, `generateInvoice`, `pettyCash`, `reconciliation`, `gMode`, `qc` | Uses nearly every operations and finance endpoint |
| aladdin | `auth`, `registration`, `businesses`, `merchant`, `business`, `location`, `wallets`, `devlopersApi`, `issueReport`, `agentMessaging`, `agentRating`, `centrifuge`, `campaign` | Most seller endpoints sit under `/businesses/:businessId/...` |
| accounts | `auth`, `invoice`, `legalOrder`, `users`, `businesses`, `departments`, `teams`, `merchants`, `reports`, `hubs` | Read and export only; CSV and PDF downloads |

Login endpoints differ: hive uses `/login/sso` and `/login/sso/verify`; accounts uses `/login/ops-otp` and `/login/ops-otp/verify`; aladdin uses phone and password with OTP at signup.

### Status enums

| Where | Name | Form |
|---|---|---|
| `hive/src/lib/constants.ts` | `TransferStatus` enum | Numbers 1 to 45 and 53 |
| `aladdin/src/lib/constants.ts` | `transferStatus` object | The same numbers |
| `src/modules/orders/types` in each app | `orderStatusSchema` (Zod) | Text values such as `received_at_lmh` |

Numbers 38 to 41 are sub-sort (38 and 39 delivery, 40 and 41 return) and 42 to 45 are reverse pickup. Commented-out lines under the hive enum show an older numbering; do not use them. Runs, baskets, invoices and hub processing have their own text statuses, listed on the [Statuses](../business/statuses.html) page.

### Money fields and formulas

Order amounts come back in taka and, for several fields, also in paisa (1/100 of a taka) with the suffix `_in_paisha`, spelled that way in code. The frontend types in `src/modules/orders/types/index.ts` (hive and aladdin) include:

| Taka field | Paisa field | Meaning |
|---|---|---|
| `collectable_amount` | `collectable_amount_in_paisha` | Cash the rider must collect ({{term "amount-to-collect"}}) |
| `collected_amount` | `collected_amount_in_paisha` | Cash actually collected ({{term "collected-amount"}}) |
| `delivery_fee` | `delivery_fee_in_paisha` | {{term "delivery-fee"}} |
| `cod_fee` | `cod_fee_in_paisha` | {{term "cod-fee"}} |
| `total_fee` | `total_fee_in_paisha` | {{term "total-fee"}}: delivery fee + COD fee + extra charges − discount |
| | `additional_charge_in_paisha`, `discount_in_paisha`, `selling_price_in_paisha` | Extras, discount and item value |

Two formulas are computed on the frontend:

- **Hub deposit** in `hive/src/modules/hub-payments/components/mutation/hub-payment-submission-form.tsx`: `payable = sumCollected − (moneyLost + cashInHand + withdrawnExpenses) + carryAmount`. The form blocks submission unless the listed transactions add up to this amount. See {{term "hub-payment-deposit"}}.
- **Delivery run cash** in `hive/src/modules/delivery-run/components/detail/`: the run detail and the cash note modal show `totalCollectedAmount − cashInHand`, named `previousCollectedAmount` in the modal.

The seller's invoice (COD collected minus fees plus compensation) is produced by the backend; the frontends only show and export it. See {{term "merchant-invoice-payment-invoice"}} and {{term "payable-vs-receivable"}}.
