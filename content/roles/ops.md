---
title: Ops and support
section: roles
order: 5
level: basics
roles: ops
summary: Where in hive to do each job, the terms and the statuses.
---
Most of your work happens in hive, the operations panel. You log in through Carrybee's single sign-on and then pick the hub you work for; the menu shows only the pages your permissions allow. If a page in this list is missing from your menu, you do not have access to it yet. Ask your manager for the permission rather than borrowing someone else's login.

## What you care about

- **Where each job is done** in hive's menu.
- **What a status means**, so you can answer "where is my parcel?" correctly.
- **The words** sellers, riders and the system use, which are not always the same.
- **Cash.** Runs must be closed and deposits submitted on time, or money goes unaccounted.

## Read in this order

1. [Courier 101](../business/courier-101.html): the basics of courier work.
2. [How Carrybee works](../business/how-carrybee-works.html): how a parcel and its money move through Carrybee.
3. [Statuses](../business/statuses.html): every status number and what it means.
4. [Courier terms](../terms.html): every word you will meet on screen, searchable.
5. [The hive map](../apps/hive.html): every hive page and what it is for.
6. [The aladdin map](../apps/aladdin.html): what the seller sees, useful when you support one.

## Where in hive do I…

Menu names are as they appear in hive's sidebar (built in `hive/src/hooks/use-menu.ts`). The chip opens that module on the hive map, which lists its pages and URLs.

| I want to… | Menu | Module |
| --- | --- | --- |
| Find an order or a parcel | Orders › List | {{module "hive" "orders"}} |
| Create an order for a seller | Orders › Create, Create Bulk Order | {{module "hive" "orders"}} |
| See sellers' pickup requests | Orders › Pickup Requests | {{module "hive" "pickup-requests"}} |
| Assign a rider to collect from stores | Run Management › Assign Pickup Agent | {{module "hive" "pickup-order"}} |
| Follow or close a pickup run | Run Management › Pickup Run | {{module "hive" "pickup-run"}} |
| Receive parcels a seller drops at the hub | Not in the sidebar; open `/hub-pickup/create` | {{module "hive" "hub-pickup"}} |
| Check in, weigh and sort parcels at the hub | Order Processing › List | {{module "hive" "order-processing"}} |
| Handle a parcel with no order ({{term "no-entry" "no-entry"}}) | Orders › No Entry Orders | {{module "hive" "no-entry-orders"}} |
| Fix an order missing zone or weight | Incomplete Orders › List | {{module "hive" "incomplete-orders"}} |
| Print missing labels (POD) | Orders › Central POD Print | {{module "hive" "central-pod-print"}} |
| Pack and track baskets | Basket › List | {{module "hive" "basket"}} |
| Send or receive a transfer run between hubs | Run Management › Transfer Run | {{module "hive" "transfer-run"}} |
| Check missing or extra parcels after receiving | Orders › Missing Orders & Baskets | {{module "hive" "missing-extra-parcels"}} |
| Sort at the central warehouse | Central Sorting › List, Unsorted | {{module "hive" "central-sorting"}} |
| Sort at a sub-sort centre | Sub Sort › List, Unsorted | {{module "hive" "order-processing"}} |
| Set up PTN racks and pigeon holes | PTN › Rack Configs; Racks › List | {{module "hive" "ptn"}} |
| See parcels waiting for delivery at my hub | Delivery Orders › List | {{module "hive" "delivery-orders"}} |
| Build a delivery run for a rider | Run Management › Delivery Run | {{module "hive" "delivery-run"}} |
| Close a run and count the cash | Run Management › Delivery Run (run detail) | {{module "hive" "delivery-run"}} |
| Force close a run the rider app never ended | Run Management › Delivery Run (run detail) | {{module "hive" "delivery-run"}} |
| Find runs still open | Invoices › Run not closed | {{module "hive" "run-not-closed"}} |
| Change an order's status by hand | Delivery Orders › Order Status Change | {{module "hive" "order-status-change"}} |
| Approve or reject a rider's urgent request | QC Panel › Urgent Approvals | {{module "hive" "qc"}} |
| Handle returns waiting at the hub | Return Orders › Pending, First Mile | {{module "hive" "return-orders"}} |
| Send returns back to sellers | Run Management › Return Run | {{module "hive" "return-run"}} |
| Collect an item from a buyer (reverse pickup) | Customer Return Orders; Run Management › Customer Return Run | {{module "hive" "customer-return-run"}} |
| Mark or review a lost parcel | Invoices › Lost Orders | {{module "hive" "lost-order"}} |
| Submit the hub's cash deposit | Payable › Hub payment | {{module "hive" "hub-payments"}} |
| See which hubs have not deposited | Invoices › Hub Payment not submitted | {{module "hive" "hub-payment-not-submitted"}} |
| Approve hub deposits (finance) | Hub Payment Approvals › List | {{module "hive" "hub-payment-approvals"}} |
| Generate and pay seller invoices (finance) | Invoices › Generate Invoice, Unpaid, Processing, Paid | {{module "hive" "generate-invoice"}} |
| Handle an early payout request | Invoices › On Demand Withdraw Requests | {{module "hive" "on-demand"}} |
| Request or approve petty cash | Petty Cash › Allocation Requests, Approval | {{module "hive" "petty-cash"}} |
| Approve rider allowance claims | Reconciliation › Allocation Requests | {{module "hive" "reconciliation"}} |
| Approve a new seller store | Pending Stores › List | {{module "hive" "pending-stores"}} |
| Verify a store's map location | Store Location Verification › List | {{module "hive" "store-location-verification"}} |
| Approve a seller's new payout account | Pending Payments › List | {{module "hive" "pending-payments"}} |
| Look up a seller's business, price plan or stores | Business › List | {{module "hive" "businesses"}} |
| Review a seller's fraud report | Business › Fraud Reports | {{module "hive" "fraud-activities"}} |
| Manage riders, applications and attendance | All Agent, Agent Application, Agent Attendance | {{module "hive" "agents"}} |
| Manage hubs, zones, areas and sectors | Hubs, Zone, Area, Sector | {{module "hive" "warehouses"}} |
| See how my hub is doing | Hubs › Hub Performance | {{module "hive" "hub-performance"}} |
| Download a consignment or basket report | Consignment wise reports | {{module "hive" "reports"}} |
| See who changed what | Activity Log › View | {{module "hive" "activity-log"}} |
| Repair a run, reset an agent's login, impersonate a user | G-Mode › Issue Resolution | {{module "hive" "g-mode"}} |

Some of these pages are filtered by the hub you picked at login. If a list looks empty, check which hub you selected at login first.

## Statuses you will use most

| Status | What to tell a seller |
| --- | --- |
| {{status 2}} | We have the request; a rider will be assigned. |
| {{status 7}} | The parcel is at our first hub. |
| {{status 35}} | It is moving between hubs. |
| {{status 13}} | It is at the hub that will deliver it. |
| {{status 14}} | It is out with a rider. |
| {{status 16}} | Delivery is paused, usually at the buyer's request, and will be tried again. |
| {{status 17}} | Delivery failed; it is coming back to you. |
| {{status 32}} | You have the parcel back. |

The full list is on [Statuses](../business/statuses.html). Words you don't recognise are in [Courier terms](../terms.html), for example {{term "urgent-approval" "urgent approval"}}, {{term "force-close" "force close"}} and {{term "cash-in-hand" "cash in hand"}}.
