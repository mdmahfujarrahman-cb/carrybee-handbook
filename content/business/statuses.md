---
title: Statuses
section: business
order: 3
level: carrybee
roles: backend, qa, ops
summary: Every order status, what it means and which app moves it.
---
Every parcel has a {{term "transfer-status" "transfer status"}}: a number that changes each time the parcel is scanned or someone acts on it. The full list, read from the code at build time, is in the table below. This section explains how to read it.

## Numbers and text

The same status appears in two forms.

| Form | Where it is defined | Who sees it |
| --- | --- | --- |
| A number, 1 to 45 plus 53 | `TransferStatus` enum in `hive/src/lib/constants.ts`, and the `transferStatus` object in `aladdin/src/lib/constants.ts` | hive screens, filters and API payloads |
| A text value such as `received_at_lmh` | `orderStatusSchema` in `src/modules/orders/types` in hive and aladdin | The merchant panel and its filters |

The text list is shorter than the number list: aladdin has no text value for each separate hop between hubs, only broader ones such as `sorted` and `in_transit`. The backend does the mapping (not confirmed in code, since the backend is not in these repos).

Numbers 46 to 52 are not used. Hive's enum also has commented-out lines with an older numbering, where 36 to 45 meant different things. Ignore those comments and trust the live enum.

## The main path

{{diagram "statuses"}}

On a normal delivery the number goes up step by step: {{status 1}}, {{status 2}}, {{status 3}}, {{status 4}}, {{status 7}}, then through sorting ({{status 9}}, {{status 11}}, or the sub-sort pair {{status 38}} and {{status 39}}), {{status 13}}, {{status 14}} and finally {{status 15}}.

Statuses 23 to 34 are the same journey in reverse for a return, ending at {{status 32}}. Statuses 42 to 45 belong to a reverse pickup from the buyer.

## Final statuses

A {{term "final-status" "final status"}} ends the parcel's journey. These include {{status 15}}, {{status 18}}, {{status 22}}, {{status 21}}, {{status 32}}, {{status 20}} and {{status 19}}. Money settles on final statuses: a seller is paid for a parcel after it is delivered, and compensation follows a lost or damaged one.

Changing a final status by hand is restricted to the {{term "g-mode"}} area in hive. If a test or a support case needs to "undo" a delivered parcel, that is where it happens. A lost parcel that turns up again moves to {{status 36}}, and a parcel that needs to go back into the flow can be set to {{status 37}}.

## Groups and tabs

The merchant panel does not show numbers. It groups statuses into four tabs ({{term "status-group-tab" "status group"}}), defined in `ORDER_STATUS_GROUP_MAP` in `aladdin/src/modules/orders/types/index.ts`:

| Tab | Text statuses in it |
| --- | --- |
| Active | `sorted`, `received_at_lmh`, `assigned_for_delivery`, `on_hold`, `in_transit`, `pickup_requested` |
| Delivered | `delivered`, `partial_delivery`, `exchange` |
| Returned | `return`, `paid_return`, `requested_for_customer_change` |
| Returning | the `return_*` chain, including `return_returned_to_merchant`, plus the four `reverse_pickup_*` statuses |

Note that a parcel already returned to the seller sits in the **Returning** tab, not **Returned**. Keep that in mind when you test the tabs or answer a seller's question.

Other things have statuses too: runs go pending, started, transferring, closed; baskets go created, printed, assigned, reached, opened, received; invoices go unpaid, processing, paid. See [How Carrybee works](how-carrybee-works.html) for where each one fits.
