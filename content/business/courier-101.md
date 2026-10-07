---
title: Courier 101
section: business
order: 1
level: basics
roles: frontend, backend, qa, product, ops
summary: Logistics basics for people new to courier work.
---
A courier company moves small packages from the person who sends them to the person who receives them. For an online shop, that means taking a parcel from the shop, getting it across the country and handing it to the buyer. This page explains the ideas every courier uses. None of it is specific to Carrybee, so you can read it without knowing anything about our code.

## The three legs of a trip

Couriers split every parcel's trip into three legs, together called the {{term "first-mile-middle-mile-last-mile" "first mile / middle mile / last mile"}}.

| Leg | From | To | What happens |
| --- | --- | --- | --- |
| First mile | The seller's shop or warehouse | The first hub | A rider collects the parcels, or the seller drops them off. |
| Middle mile | One hub | Another hub | Parcels travel in bulk, often by truck, towards the buyer's area. |
| Last mile | The hub nearest the buyer | The buyer's door | A rider delivers the parcel and, often, collects payment. |

The first and last mile are done by riders on motorbikes or bicycles, one stop at a time. The middle mile is where parcels move in volume. The last mile is the most expensive leg, because each delivery is a separate stop and many go wrong: the buyer is not home, the address is wrong, or the buyer changes their mind.

## Hubs and sorting

A {{term "hub" "hub"}} is a branch office with a sorting floor. Parcels arrive at a hub, get checked in, and are grouped by where they need to go next. That grouping is called sorting.

Sorting is the reason a courier can be cheap. Instead of driving each parcel straight to its buyer, the courier collects hundreds of parcels in one area, sorts them by destination, and sends each group together. A large network often has a central sorting centre where parcels from many hubs meet, plus smaller regional sorting centres.

Parcels for the same next stop are packed into a sealed bag or crate, so a hub can hand over fifty parcels as one unit and the next hub can check that nothing is missing. Hubs scan every parcel in and out. A scan is how the system knows where the parcel is.

## Runs and riders

Riders work in trips. A trip is usually called a run: one rider, one list of stops, one start and one end.

- A **pickup run** visits sellers and collects parcels.
- A **delivery run** takes parcels out to buyers.
- A **transfer run** moves bags of parcels between hubs.
- A **return run** takes failed parcels back to the seller.

At the end of a delivery run the rider returns to the hub, hands back anything not delivered and hands over the cash collected. The hub counts it and closes the run. A run that is not closed means cash and parcels that nobody has accounted for yet.

## Cash on delivery

In Bangladesh many online buyers pay in cash when the parcel arrives. This is {{term "cod"}}, cash on delivery. The courier collects the money on behalf of the seller.

COD makes the courier responsible for a lot of cash. Every taka has to be traced from the buyer's hand to the rider, from the rider to the hub, from the hub to the company's bank account, and finally to the seller. The courier keeps its fees along the way: a delivery fee per parcel and a small COD fee for handling the cash. The seller gets the rest in a regular payout, with an invoice that shows the sums.

## When delivery fails

Not every parcel reaches its buyer. The buyer may refuse it, be unreachable, or want only part of the order. When that happens the parcel goes back to the seller, travelling the same legs in reverse. This is a **return**.

Couriers also handle trips that start at the buyer. A **reverse pickup** collects an item from the buyer and brings it back to the seller, for example a product the buyer wants to send back. An **exchange** does both in one visit: deliver a new item and take the old one. All of this is called reverse logistics. It costs the courier as much work as a delivery and usually earns less, so return rates matter to everyone.

Sometimes a parcel is damaged or lost. The courier then pays the seller compensation, usually based on the item's declared value.

## Why statuses exist

A parcel changes hands many times: seller, rider, hub, truck, another hub, another rider, buyer. The courier records each step as a status, such as "picked up", "at hub", "out for delivery" or "delivered". A status answers three questions at once: where is the parcel, who is responsible for it right now, and what should happen next.

Statuses drive almost everything you see on screen. Lists of work are filtered by status ("all parcels waiting to be sorted"). Sellers track their orders by status. Money moves only when certain statuses are reached; a seller is paid for a parcel only after it is delivered. Some statuses are final, which means the parcel's journey is over and the status should not change again.

:::note
**Know this already?** Go to [How Carrybee works](how-carrybee-works.html). It covers the same ground with Carrybee's own words, apps and status numbers.
:::
