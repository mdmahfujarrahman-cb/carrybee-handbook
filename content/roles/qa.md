---
title: QA and testing
section: roles
order: 3
level: carrybee
roles: qa
summary: End-to-end journeys, roles and permissions, statuses and known bugs.
---
None of the three apps has automated tests, so manual and end-to-end testing is the main check on every release. A single parcel crosses all three apps, and the rider's part happens in a mobile app that is not in these repos. Good test cases follow one order from seller to payout and check the status and the money at each step.

## What you care about

- **Journeys that cross apps.** An order booked in aladdin is processed in hive and invoiced through hive, aladdin and accounts. Bugs often sit at the hand-offs.
- **Statuses.** Each step must leave the parcel in the right status number, and the seller must see the right tab.
- **Money.** COD collected, fees, hub deposits and invoices must add up.
- **Permissions.** Each role must see its pages and nothing else.
- **Exception paths.** Returns, partial deliveries, urgent approvals and lost parcels are where real users get stuck.

## Read in this order

1. [Courier 101](../business/courier-101.html) if logistics is new to you.
2. [How Carrybee works](../business/how-carrybee-works.html): the 13-step journey you will test.
3. [Statuses](../business/statuses.html): numbers, final statuses and merchant tabs.
4. The app maps, A-to-Z view: [aladdin](../apps/aladdin.html), [hive](../apps/hive.html), [accounts](../apps/accounts.html).
5. [Known issues](../known-issues.html): bugs already found, with file and line.
6. [Ops and support](ops.html): where each job is done in hive.
7. [Courier terms](../terms.html) whenever a word on screen is unclear.

## Happy path

Run this with one fresh seller account and one order, and check the status after each step.

| # | Step | App and place | Check |
| --- | --- | --- | --- |
| 1 | Sign up with a phone number, verify OTP, create a business | aladdin register, create business | Business appears in the business chooser |
| 2 | Add a pickup store and a payout account | aladdin stores, payments | Store waits for ops approval in hive pending stores; payout change waits in pending payments |
| 3 | Book an order with a COD amount and print the label | aladdin orders | {{status 1}}; consignment ID shown; label prints |
| 4 | Request a pickup | aladdin request pickup | {{status 2}} |
| 5 | Assign the pickup to a rider's run | hive pickup orders, pickup run | {{status 3}}, then {{status 4}} after pickup |
| 6 | Receive and sort at the first-mile hub | hive order processing | {{status 7}} |
| 7 | Pack a basket and send a transfer run; receive it | hive baskets, transfer run | {{status 8}} to {{status 13}}; no missing or extra parcels |
| 8 | Add to a delivery run | hive delivery orders, delivery run | {{status 14}}; seller sees it in the Active tab |
| 9 | Deliver (rider app or test data) and close the run with the cash count | hive delivery run | {{status 15}}; cash note matches COD |
| 10 | Submit and approve the hub deposit | hive hub payment, hub payment approvals | Payable amount follows the formula on [How Carrybee works](../business/how-carrybee-works.html#money-flow) |
| 11 | Generate, process and pay the invoice | hive invoices | Invoice moves unpaid, processing, paid |
| 12 | Seller sees the invoice; finance exports it | aladdin invoices; accounts invoice | Amount = COD collected − fees |

## Failure paths

| Case | How to trigger | Expect |
| --- | --- | --- |
| Return | Delivery fails, return approved | {{status 17}}, then 23 to 31 back through the network, then {{status 32}} via a return run |
| Partial delivery | Buyer keeps part of the order | {{status 18}}; needs urgent approval; collected amount differs from amount to collect |
| Exchange | Exchange order delivered | {{status 22}}; old item comes back |
| Paid return | Buyer still pays the delivery charge on a return | {{status 21}}; needs urgent approval. Meaning read from the name, not confirmed in code |
| On hold | Buyer asks for another day | {{status 16}}; parcel goes out again |
| Reverse pickup | Seller creates one in aladdin orders, reverse pickup | {{status 42}} to {{status 45}}; hive customer return run |
| Lost | Mark lost in hive lost orders | {{status 20}}; compensation charged to rider, hub or Carrybee; {{status 36}} if found |
| Urgent approval, seller | Rider requests a return or price change; seller answers in aladdin high alert | Approve and reject both work and change the outcome. Reject is a known suspect, see below |
| Urgent approval, QC | Same request answered in hive QC panel | Approved or rejected; fake requests tracked |
| No-entry | Scan a parcel with no order at a hub | Appears in hive no-entry orders and in the seller's high alert |
| Incomplete order | Order missing zone or weight | Appears in hive incomplete orders |
| Force close | Delivery run whose end event never arrived | Manager closes it with remarks; it leaves run-not-closed |
| Missed hub deposit | Hub does not submit a deposit | Shows in hub payment not submitted |

## Who to log in as

| App | Role | Notes |
| --- | --- | --- |
| aladdin | Business owner | Full access to their businesses |
| aladdin | Staff with role operator, and staff with role accountant ({{term "operator" "operator"}}) | Added by the owner; check what each can see |
| hive | Hub manager | Scoped to one hub; picks the hub after SSO |
| hive | KAM, QC reviewer, finance, audit | Each sees a different part of the menu |
| hive | Super admin with {{term "g-mode"}} | Repair tools and impersonation; use sparingly in tests |
| accounts | Revenue recognition | Key `invoice_and_billing.revenue_recognition.view`; invoices pages |
| accounts | Legal panel | Key `invoice_and_billing.legal_panel.view`; legal order export |

hive and aladdin both support {{term "impersonate" "impersonate"}}, which lets support staff log in as another user to reproduce a problem. For permission tests, open a page's URL directly as a user without the key: the proxy should redirect, and the menu item should not appear.

## Bugs to keep in mind

- In aladdin, rejecting an urgent approval calls the same URL as approving it. Test that reject really rejects.
- In accounts, the consignment report and the merchant report call the same endpoint. Compare the two exports.
- Some aladdin `APP_ROUTES` have no page, and `update-password` 404s on a hard refresh.
- In the aladdin order tabs, a parcel returned to the seller sits under **Returning**, not **Returned** (see [Statuses](../business/statuses.html)). Confirm with product whether that is intended.

Details, file and line for each are on [Known issues](../known-issues.html).
