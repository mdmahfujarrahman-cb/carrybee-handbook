---
title: Product and design
section: roles
order: 4
level: carrybee
roles: product
summary: Who uses each app, the main journeys and the exception flows.
---
Carrybee has three very different user groups: sellers running a shop, staff running a physical network, and finance and legal staff exporting records. Each app is shaped by its users' day. The fastest way to understand them is to follow one parcel through all three, then look at what happens when delivery goes wrong, because that is where most of the work and most of the cost sit.

## What you care about

- **Who the users are** and what job each app does for them.
- **The main journeys**, from a seller signing up to getting paid, and from building the network to paying the seller.
- **Exception flows**: returns, partial deliveries, urgent approvals, lost parcels. They cost money and they generate support work.
- **Metrics** that tell you whether the network is healthy.

## Read in this order

1. [Courier 101](../business/courier-101.html) if logistics is new to you.
2. [How Carrybee works](../business/how-carrybee-works.html): business model, users and the 13-step journey.
3. The aladdin map, A-to-Z view: [aladdin](../apps/aladdin.html), the seller's journey in 8 stories.
4. The hive map, A-to-Z view: [hive](../apps/hive.html), the staff journey in 14 stories.
5. The accounts map, A-to-Z view: [accounts](../apps/accounts.html), finance and legal in 5 stories.
6. [Statuses](../business/statuses.html): what sellers see versus what staff see.
7. [Courier terms](../terms.html): the vocabulary used on screen.
8. [Known issues](../known-issues.html): bugs that affect users today.

## Users and their jobs

| App | Users | Main jobs |
|---|---|---|
| aladdin | Seller (business owner); staff they add as operator or accountant ({{term "operator"}}); large sellers using the developer API | Book orders (single, bulk, photo, API), request pickups, track parcels, answer the {{term "high-alert"}} queue, chat with the rider, read invoices, withdraw early, pay dues. English and Bangla. |
| hive | Hub managers, KAMs, QC reviewers, finance, audit, HR, admins, super admins | Build the network, onboard sellers, run pickups, sorting, transfers and deliveries, approve exceptions, count and deposit cash, invoice and pay sellers, approve expenses, fix problems in G-Mode. |
| accounts | Finance and legal staff | Export merchant invoices for {{term "revenue-recognition"}}, export orders by parcel type for the {{term "legal-panel-legal-order"}}, check sellers' identity status, download consignment reports. |

## Journeys

Each app map has an A-to-Z view that tells its journey as short stories, with the modules involved.

| App | Journey | Stories |
|---|---|---|
| [aladdin](../apps/aladdin.html) | From a seller signing up to getting paid | Join Carrybee, set up the shop, book orders, hand parcels to Carrybee, track the parcel, handle problems, after delivery, get paid |
| [hive](../apps/hive.html) | From building the network to paying the seller | Build the network, win a seller, order created and collected, first-mile hub, middle mile, last mile, review and quality, when delivery fails, close the day, cash comes in, pay the seller, seller pays Carrybee, running costs, watch and fix |
| [accounts](../apps/accounts.html) | From signing in to exporting invoices and reports | Sign in, know the sellers, recognise revenue, legal review, report |

## Exception flows and what they cost

| Flow | What happens | Cost to the business |
|---|---|---|
| Return ({{status 17}} to {{status 32}}) | Parcel travels back through every leg to the seller | Network work twice with no delivery; a {{term "return-fee"}} recovers part of it |
| Partial delivery ({{status 18}}) | Buyer keeps part; the rest is returned | Less COD collected; needs an urgent approval |
| Exchange ({{status 22}}) | New item delivered, old item collected in one visit | Extra handling; an exchange fee applies |
| Reverse pickup ({{status 42}} to {{status 45}}) | Rider collects an item from the buyer | A trip with no COD; charged as an extra service |
| On hold ({{status 16}}) | Buyer asks for another day | Another delivery attempt; extra attempts can be charged ({{term "attempt"}}) |
| Urgent approval | Rider asks to change the outcome at the door; seller or QC decides | Rider waits; slow answers delay the run. QC also tracks fake requests ({{term "fake-request"}}) |
| Lost ({{status 20}}) or damaged ({{status 19}}) | Parcel cannot be delivered | {{term "compensation"}} to the seller, charged to the rider, hub or Carrybee |
| No-entry | Parcel at a hub with no order | Parcel is stuck until ops create, process or return it |
| Run not closed / force close | Rider app never sent the run's end event | Cash is unaccounted until a manager closes the run |
| Fraud report | Seller reports a buyer for refusing COD or abusing returns | Ops review; the fraud checker shows a phone number's history |

## Metrics

| Metric | What it tells you | Where it shows |
|---|---|---|
| {{term "delivery-success-rate"}} | Share of parcels delivered | hive {{module "hive" "hub-performance"}} and {{module "hive" "agents"}}; per buyer phone in aladdin {{module "aladdin" "customers"}} (fraud checker) and on the order form |
| {{term "return-rate-hold-rate"}} | Share returned or put on hold; high values point to bad buyers, addresses or riders | hive {{module "hive" "hub-performance"}}, {{module "hive" "business-insights"}} |
| {{term "aging"}} and {{term "backlog-pipeline"}} | How long parcels sit at a stage; old parcels mean a backlog | hive {{module "hive" "hub-performance"}} |
| {{term "run-completion"}} | Runs closed and parcels per rider | hive {{module "hive" "hub-performance"}} |
| {{term "agent-rating"}} | Buyers' and sellers' scores for riders | hive {{module "hive" "rating-management"}}; aladdin {{module "aladdin" "agent-rating"}} |
| {{term "churned-merchant"}} | Sellers who stopped sending parcels | hive {{module "hive" "business-insights"}} |
| Seller's own numbers | Order trends, overall statistics, top stores | aladdin {{module "aladdin" "analytics"}}, {{module "aladdin" "dashboard"}} |

The exact definitions behind each number are computed by the backend. Ask the backend team before you rely on one for a decision.
