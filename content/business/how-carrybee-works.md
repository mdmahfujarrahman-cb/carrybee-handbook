---
title: How Carrybee works
section: business
order: 2
level: carrybee
roles: frontend, backend, qa, product, ops
summary: The business model, the three apps and a parcel's journey from A to Z.
---
Carrybee Express picks up parcels from online sellers, moves them through its own hub network, delivers them to buyers anywhere in Bangladesh, collects {{term "cod"}} at the door, keeps its fees and pays the rest to the seller. This page follows one parcel and its money through the three apps. If logistics is new to you, read [Courier 101](courier-101.html) first.

:::carrybee
These are the places where Carrybee's words and rules differ from what you may expect.

- **POD means the label.** In this codebase {{term "pod-label"}} is the printed parcel label with barcode and QR code ("POD print"), not proof of delivery.
- **QC is approval review.** The {{term "qc-reviewer"}} team reviews riders' urgent approval requests and looks for fake ones. It is not a physical check of the parcel.
- **Urgent approval.** When a rider wants to change an outcome at the door (return, price change, exchange, partial delivery, paid return), the change needs an {{term "urgent-approval" "urgent approval"}} from the seller in aladdin or from QC in hive.
- **PTN racks.** Sub-sort centres use racks with 12 or 24 pigeon holes ({{term "ptn-rack"}}), each mapped to a hub or linehaul.
- **Hub deposit formula.** A hub's deposit is payable = sum collected − (money lost + cash in hand + withdrawn expenses) + carry. See {{term "hub-payment-deposit" "hub payment"}}.
- **ISD / OSD pricing.** Prices, COD fees and return fees depend on the zone: {{term "isd-osd-sub-area"}}.
- **Amounts in paisa.** Order money fields also exist in paisa, 1/100 of a taka, with an `_in_paisha` suffix. See {{term "paisa-in-paisha" "paisa"}}.
- **Sorting window.** A day's sorting numbers count from 18:00 to 06:00 the next morning. See {{term "pickup-sorting-window" "pickup / sorting window"}}.
- **Statuses 38 to 41 are sub-sort, 42 to 45 are reverse pickup.** {{status 38}} and {{status 39}} are deliveries going to and at a sub-sort centre, 40 and 41 are returns. {{status 42}} to {{status 45}} cover a reverse pickup from waiting to on hold.
:::

## The business model

Carrybee's customers are sellers. The code calls a seller a merchant ({{term "merchant" "merchant"}}). One merchant login can own several businesses ({{term "business" "business"}}), and each business has its own pickup stores, price plan, invoices and payout account. Large sellers get a key account manager ({{term "kam"}}) who handles their pricing and pickups.

Carrybee earns money per parcel. A seller's price plan ({{term "price-plan" "price plan"}}) sets the fees:

| Fee | What it is |
| --- | --- |
| {{term "delivery-fee"}} | Charge per parcel, by destination zone and weight. |
| {{term "cod-fee"}} | Charge for handling the cash, a percentage of the amount collected. |
| {{term "return-fee"}} | Charge for bringing a failed parcel back, a percentage of the delivery fee. |
| {{term "additional-charge"}} | Extras such as packaging, more delivery attempts or reverse pickup. |

Prices depend on destination and weight (see {{term "location-slab-weight-slab" "location slab / weight slab"}}) and are in taka (৳, BDT). For a COD order, Carrybee subtracts its fees from the cash it collected and pays the seller the difference. A business is prepaid, postpaid or corporate ({{term "prepaid-postpaid-corporate" "prepaid / postpaid / corporate"}}); a postpaid business is billed later and can end up owing Carrybee money.

## Who uses which app

| App | Users | What they do | Login |
| --- | --- | --- | --- |
| [aladdin](../apps/aladdin.html) | Sellers and the staff they add (see {{term "operator" "operator"}}) | Sign up, add stores and a payout account, book orders, request pickups, track parcels, answer exceptions, see invoices and pay dues. English and Bangla. | Phone and password, OTP at signup. Carrybee staff can impersonate a seller. |
| [hive](../apps/hive.html) | Carrybee staff: hub managers, KAMs, QC, finance, audit, admins | Run the network: hubs, riders, pickup and delivery runs, sorting, baskets, linehauls, returns, QC, cash deposits, invoices, petty cash, allowances. | {{term "sso"}} through the backend, then pick a hub. Access by permission keys. Support can impersonate. |
| [accounts](../apps/accounts.html) | Finance and legal staff | Read and export: revenue recognition invoices, legal order export, business and merchant lists, consignment reports. | Phone and OTP. Two permissions: revenue recognition and legal panel. |

Riders, called {{term "agent-rider" "agent"}} in code, use a separate mobile app that is not in these repos. Anything a rider does on the road shows up in the three panels only as a new status.

## A parcel from A to Z

The order of steps follows the `TransferStatus` numbers in `hive/src/lib/constants.ts`. Each step says who does it, where it lives in code and which status it leaves behind. Every status is listed on the [Statuses](statuses.html) page.

### Set up

**1. Seller signs up and creates a business** (aladdin). The seller registers with a phone number, verifies an OTP and creates or picks a business. They add stores ({{term "store" "store"}}), which are pickup and return addresses tied to a hub and a route, and a payout method: a bank account or a mobile wallet. {{module "aladdin" "authentication"}} {{module "aladdin" "businesses"}} {{module "aladdin" "stores"}} {{module "aladdin" "payments"}}

### First mile

**2. Seller books orders** (aladdin). Orders come from a single form, a spreadsheet ({{term "bulk-order" "bulk order"}}), a photo read by OCR, or the developer API. Each order gets a {{term "consignment-id-cid" "consignment ID (CID)"}} and an {{term "amount-to-collect" "amount to collect"}}. The seller prints the POD label for each parcel. Status: {{status 1}}. {{module "aladdin" "orders"}}

**3. Seller requests a pickup** (aladdin). A {{term "pickup-request" "pickup request"}} says "collect N parcels from store X". A KAM can also create one from hive for a key account. Status: {{status 2}}. {{module "aladdin" "pickup-requests"}} {{module "hive" "pickup-requests"}}

**4. Hub assigns a pickup run** (hive). The hub groups stores by route and adds them to a rider's {{term "pickup-run" "pickup run"}}. A seller can instead drop parcels at the hub, confirmed by OTP ({{term "hub-pickup-drop-off" "hub pickup (drop-off)"}}). Status: {{status 3}}. {{module "hive" "pickup-order"}} {{module "hive" "pickup-run"}} {{module "hive" "assign-run"}} {{module "hive" "hub-pickup"}}

**5. Rider collects the parcels** (rider app). This happens outside these repos; the run is closed back in hive. Status: {{status 4}}, then {{status 7}} when the parcel reaches the hub.

**6. First-mile hub processes them** (hive). Staff weigh and sort parcels and print missing labels. Exceptions get their own queues: a {{term "no-entry" "no-entry"}} parcel is physically at the hub with no order in the system, and an {{term "incomplete-order" "incomplete order"}} is missing data such as zone or weight. {{module "hive" "order-processing"}} {{module "hive" "no-entry-orders"}} {{module "hive" "incomplete-orders"}} {{module "hive" "central-pod-print"}}

### Middle mile

**7. Pack baskets and move them** (hive). Sorted parcels go into a {{term "basket" "basket"}}, a sealed bag. A {{term "transfer-run" "transfer run"}} carries baskets along a {{term "linehaul" "linehaul"}} with a printed gate pass. The receiving hub scans the basket and the system flags missing or extra parcels ({{term "missing-extra-parcel" "missing / extra parcel"}}). Status: {{status 8}} to {{status 11}}, and {{status 35}} while in transit. {{module "hive" "basket"}} {{module "hive" "transfer-run"}} {{module "hive" "linehaul"}} {{module "hive" "missing-extra-parcels"}}

**8. Central sorting and sub-sort** (hive). The {{term "central-warehouse" "central warehouse"}} (hub IDs 71 and 72 in code) and regional {{term "sub-sort" "sub-sort"}} centres route each parcel to its {{term "lmh-last-mile-hub"}}. Status: {{status 38}} and {{status 39}} at sub-sort, then {{status 12}} and {{status 13}} at the last-mile hub. {{module "hive" "central-sorting"}} {{module "hive" "ptn"}}

### Last mile

**9. Last-mile hub builds a delivery run** (hive). Parcels are scanned into a rider's {{term "delivery-run" "delivery run"}}. Staff can reassign parcels, move them between zones, or mark them lost or damaged. Status: {{status 14}}. {{module "hive" "delivery-orders"}} {{module "hive" "delivery-run"}}

**10. Rider delivers and collects cash** (rider app). The outcome is one of {{status 15}}, {{status 16}}, {{status 18}}, {{status 22}}, {{status 21}} or {{status 17}}. Many outcomes need an urgent approval, from the seller in aladdin's high alert queue or from a QC reviewer in hive. {{module "aladdin" "high-alert"}} {{module "hive" "qc"}}

**11. Delivery run is closed** (hive). The rider hands over the cash and the hub counts it note by note ({{term "cash-note" "cash note"}}). If the rider app never sent its end event, a manager can {{term "force-close" "force close"}} the run with remarks. Runs left open appear in the run-not-closed list. {{module "hive" "delivery-run"}} {{module "hive" "run-not-closed"}}

### When delivery fails

**12. Return, reverse pickup or lost** (hive and aladdin). A return travels back through the network ({{status 23}} to {{status 31}}) and a {{term "return-run-rtm" "return run (RTM)"}} hands it to the seller: {{status 32}}. A {{term "reverse-pickup" "reverse pickup"}} fetches an item from the buyer ({{status 42}} to {{status 45}}). The seller can also {{term "send-to-another-customer" "send to another customer"}}. A lost parcel ({{status 20}}) gets {{term "compensation" "compensation"}} charged to the rider, the hub or Carrybee. {{module "hive" "return-run"}} {{module "hive" "customer-return-run"}} {{module "hive" "return-orders"}} {{module "hive" "lost-order"}} {{module "aladdin" "high-alert"}}

### Settle

**13. Seller gets paid** (hive, aladdin, accounts). Finance generates the seller's invoice (COD collected minus delivery and COD fees), sends it to the bank in bulk and marks it paid. The seller sees it in aladdin; finance and legal export it from accounts. {{module "hive" "generate-invoice"}} {{module "aladdin" "invoices"}} {{module "accounts" "invoice"}}

## Money flow

Cash is collected in the field and moves upward until it is split between Carrybee and the seller. Most of hive's finance modules check one link in this chain.

| Step | What happens | Where |
| --- | --- | --- |
| 1. Field | The rider collects COD. It is counted note by note when the delivery run closes. | {{module "hive" "delivery-run"}} |
| 2. Hub | The {{term "hub-ledger" "hub ledger"}} tracks each hub's {{term "cash-in-hand" "cash in hand"}} and {{term "carry-carry-amount" "carry"}}. | {{module "hive" "hub-ledger"}} |
| 3. Deposit | The hub deposits the cash with proof. Missed deposits are flagged. | {{module "hive" "hub-payments"}} {{module "hive" "payable-consignments"}} {{module "hive" "hub-payment-not-submitted"}} |
| 4. Approve | Finance approves deposits, one by one or in bulk. | {{module "hive" "hub-payment-approvals"}} |
| 5. Invoice | The merchant invoice is generated (regular or positive balance) and moves unpaid, processing (bank Excel, {{term "rtgs"}}), paid. | {{module "hive" "generate-invoice"}} {{module "hive" "unpaid-invoice"}} {{module "hive" "processing-invoice"}} {{module "hive" "paid-invoice"}} |
| 6. Payout | The seller is paid on a regular, custom or on-demand schedule, to a bank or an {{term "mfs"}} wallet. | {{module "hive" "wallet-provider"}} {{module "hive" "on-demand"}} {{module "aladdin" "invoices"}} |

The hub deposit in step 3 uses this formula, computed in `hive/src/modules/hub-payments/components/mutation/hub-payment-submission-form.tsx`:

```text
payable = sum collected − (money lost + cash in hand + withdrawn expenses) + carry
```

The form only lets the hub submit when its listed transactions add up to that payable amount.

Money also flows the other way. Prepaid and postpaid businesses can owe Carrybee, which shows up as a {{term "receivable-invoice" "receivable invoice"}}. Sellers pay these through bKash, Nagad, SSLCommerz or bank transfer from aladdin. On the cost side, hive tracks {{term "petty-cash" "petty cash"}}, hub {{term "rent" "rent"}} and rider {{term "delivery-allowance" "delivery allowance"}}. Each goes through a multi-step approval called {{term "reconciliation" "reconciliation"}} before the money is released by OTP. {{module "hive" "receivable-postpaid-invoices"}} {{module "hive" "petty-cash"}} {{module "hive" "reconciliation"}} {{module "hive" "delivery-allowance"}}

## Network and geography

Two trees describe the physical network. Most ops screens filter by one of their levels, and a hive user is scoped to a hub, cluster or region.

| Organisation tree | Delivery-area tree |
| --- | --- |
| Region (with a region lead) | City |
| Cluster (with a cluster lead) | Zone, which belongs to a hub |
| Hub: type ISD, OSD or sub area; tier big, mid or small; sorting hub or not | Area |
|  | Sector |

A {{term "route" "route"}} is a hub's set of zones plus a default rider, for pickup, delivery or reverse pickup. Linehauls connect hubs to sorting centres. Zone shapes are drawn in the {{term "geonix"}} map tool.

In code this lives in {{module "hive" "warehouses"}}, which is the most imported module in hive, plus {{module "hive" "region"}}, {{module "hive" "cluster"}}, {{module "hive" "route"}}, {{module "hive" "linehaul"}} and {{module "hive" "geonix"}}.
