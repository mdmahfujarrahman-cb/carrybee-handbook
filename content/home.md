---
title: Carrybee in 60 seconds
section: home
order: 0
level: basics
roles: frontend, backend, qa, product, ops
summary: What Carrybee does, in one breath, and where to start reading.
---
Carrybee Express is a parcel courier in Bangladesh. It picks up parcels from online sellers, sorts them through its hubs and delivers them to buyers, usually collecting the price in cash at the door ({{term "cod"}}). Carrybee keeps its fees and pays the rest to the seller.

Three web apps run the business. Sellers use **aladdin** to book orders, request pickups and track parcels. Carrybee staff use **hive** to run hubs, riders, sorting, returns, cash and payouts. Finance and legal staff use **accounts** to export invoices and reports. All three call one shared backend API that lives outside these repos.

A parcel's trip has three legs, the {{term "first-mile-middle-mile-last-mile" "first mile / middle mile / last mile"}}: from the seller to the nearest hub, hub to hub, and from the buyer's nearest hub to the door. Every scan moves it to a new numbered status, from {{status 1}} to {{status 15}} on a good day.

Money flows back. The rider hands the cash to the hub, the hub deposits it, finance approves the deposit and invoices the seller for the cash collected minus Carrybee's fees, then pays the seller by bank or a mobile wallet such as bKash.
