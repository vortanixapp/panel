# Tariffs

[English](tariffs.md) · [Русский](tariffs.ru.md)

This document answers one question: what resources to put into a tariff and what
the game server actually gets from them. It covers every field of the form, both
billing models, what the resources turn into on the node, how the price is
calculated and what happens when a tariff changes.

Panel section: **Administration → Tariffs**. A tariff belongs to a game; the
location is optional.

- [What a tariff decides](#what-a-tariff-decides)
- [Tariff fields](#tariff-fields)
- [Two billing models](#two-billing-models)
- [What the resources become on the node](#what-the-resources-become-on-the-node)
- [Which numbers to set](#which-numbers-to-set)
- [Ranges: when the customer chooses](#ranges-when-the-customer-chooses)
- [How many servers fit on a node](#how-many-servers-fit-on-a-node)
- [Periods and discounts](#periods-and-discounts)
- [Changing the tariff and the resources](#changing-the-tariff-and-the-resources)
- [Tariffs generated from the catalog](#tariffs-generated-from-the-catalog)
- [Common mistakes](#common-mistakes)

## What a tariff decides

A tariff settles three things at once:

1. **The price** — the monthly cost and how it depends on the resources;
2. **What the container gets** — memory, CPU, disk, slots;
3. **What the customer may change** — the sliders at checkout and when changing
   the resources of a running server.

Resources reach a server once, at handover: the customer's order is written into
the server's `limits` field and travels from there to the node. Editing a tariff
afterwards does not touch servers already handed over — they keep their limits
until their owner changes the tariff or the resources.

## Tariff fields

**Basic information**

| Field | Meaning |
|---|---|
| Tariff name | what the customer sees in the list |
| Game | required, a tariff always belongs to one game |
| Location | optional: empty means every location, set means that one only |
| Tariff type | pay for resources, or pay for slots |
| MySQL version | a note on the tariff: `mysql80`, `mysql57` or `mariadb` (see below) |
| Position | order in the list, lower goes first |
| Availability | a disabled tariff disappears from the catalog but stays on servers already using it |

**Container limits** — what the server gets:

| Field | Unit | What it does |
|---|---|---|
| CPU cores | cores | a hard CPU ceiling |
| CPU shares | share | soft CPU splitting, available only when cores are `0` |
| RAM | GB | a hard memory ceiling, at least 1 |
| Disk | GB | a quota on the server folder, at least 1 |
| Min and max slots | players | slot bounds, required for a slot tariff |

**Prices** — the set depends on the billing type (below). **Ranges** — minimum,
maximum and step for CPU, memory and disk. Rental and renewal **periods**,
**discounts**, the Anti-DDoS flag and its price.

There is no separate "monthly price" field: the panel works it out from the
prices and the smallest configuration and shows it as "Estimate: minimal
configuration". That is the "from" price the customer sees in the list.

A note on MySQL: the version chosen in the tariff is stored and shown on the
tariff card, but it does not drive database handover yet — the database engine is
picked when the database is created on the server itself and comes from the
location settings. Treat the field as a note to yourself, not a switch.

## Two billing models

**Pay for resources**

```
month = base price
      + cores   × price per core
      + GB RAM  × price per GB of RAM
      + GB disk × price per GB of disk
      + Anti-DDoS if the customer enabled it
```

Leave every per-resource price at zero and fill in only the base price, and the
tariff becomes a plain fixed one: the resources are set but do not affect the
cost. This is the normal, most common shape for boxed Start / Optimal / Maximum
tariffs.

**Pay for slots**

```
month = slots × price per slot + Anti-DDoS if the customer enabled it
```

The base price is forced to zero in this mode. The chosen number of slots is
written into the game settings (`max-players`, `maxplayers`, `MaxPlayers`,
depending on the game) and becomes read-only for the customer, labelled "Limited
by the tariff". With resource billing there is no such lock — the player count
stays in the customer's hands.

Anti-DDoS is a line on the invoice: the flag lets the customer enable the option
and the price is added to the monthly cost. The panel does not configure any
protection itself; that comes from the node's hosting provider.

## What the resources become on the node

The agent starts the container and turns the limits into Docker parameters:

| Tariff field | On the node | What happens when exceeded |
|---|---|---|
| RAM, GB | `docker run -m <GB×1024>m` | the kernel kills the server process out of memory (OOM) |
| CPU cores > 0 | `--cpus <cores>` | the server never gets more than that many cores and slows down |
| CPU cores = 0 with shares | `--cpu-shares <value>` | no ceiling; the share only matters when the CPU is contended |
| Disk, GB | a project quota of the file system | writes stop, the server reports "no space left" |
| Slots | `max_players` in the game settings | extra players cannot connect |

Three consequences worth knowing.

**Memory is a hard ceiling, not a wish.** Set it below what the game needs and
the server will die without a clear error: the container is killed by the
kernel. Follow the game's requirements, not the wish to sell cheaper.

**Disk is only capped when quotas are on.** The agent applies the quota with
`setquota` on the server's directory project, and that works when the data
directory is mounted with `prjquota`. Such a mount is prepared by the "Quotas"
step when a location is connected: the panel builds an ext4 image with project
quotas, copies the existing server files into it and mounts it over the servers
directory, adding it to `fstab`. The image size comes from the location's
`quota_size` setting and defaults to 80 % of the free space.

Two things follow. If the "Quotas" step was skipped or failed, the agent logs
"disk quotas unavailable" and any server can fill the node's disk; the state
shows up in the location diagnostics as the `quota` check. And when quotas are
on, the disk sold by all tariffs on that location is bounded by the size of the
image — you cannot hand out more space than it holds.

**Cores and shares are different things.** `--cpus 2` means "never more than two
full cores". Shares cap nothing: while the neighbours idle the server takes the
whole CPU, and under contention it gets a slice proportional to the value.
Shares make sense on oversubscribed nodes, where fairness matters more than a
ceiling. The minimum value is 2, and it can only be set when cores are `0`.

One special case: the Minecraft Java image receives `MEMORY` 256 MB below the
container limit, and never less than 512 MB. The JVM needs that headroom outside
the heap — without it the container is killed exactly when the server warms up.

## Which numbers to set

Start from the game requirements in the built-in catalog. The panel knows the
minimum and the recommended memory, the minimum disk and the minimum cores for
every game; the same values are used when a server is created without a tariff.

| Game | Minimum RAM | Recommended RAM | Minimum disk | Minimum cores |
|---|---|---|---|---|
| SA-MP | 128 MB | 256 MB | — | 0.5 |
| PocketMine-MP | 512 MB | 1 GB | 1 GB | 1 |
| Minecraft Bedrock | 512 MB | 1 GB | 1 GB | 1 |
| Minecraft Java | 1 GB | 2 GB | 2 GB | 1 |
| Garry's Mod | 1 GB | 2 GB | — | 1 |
| CS2 | 2 GB | 3 GB | — | 2 |
| Valheim | 2 GB | 4 GB | — | 2 |
| ARK: Survival Evolved | 8 GB | 12 GB | 25 GB | 4 |
| Rust | 8 GB | 16 GB | 20 GB | 4 |
| Palworld | 8 GB | 16 GB | — | 4 |

The full list lives in the panel's game catalog and is the same in every
installation.

The working rule: **take the recommended value, not the minimum**. The minimum
means "it starts and survives while empty", the recommendation means "it holds a
normal game". For disk, plan one and a half to two minimums: worlds grow, and
backups and uploaded files share the same quota.

Take cores from the game minimum rather than handing one core to everyone: light
servers such as SA-MP, Minecraft and PocketMine are fine with one, shooters and
Valheim want two, and heavy survival games — ARK, Rust, Palworld — want four.
Going far above the recommendation buys little, because game servers almost
always sit on a single thread. When a node holds many servers that idle most of
the time, `0` cores with shares is fairer than a ceiling for everybody.

Slots: some games carry their own maximum in the catalog — 64 for CS2, 32 for
Palworld, 10 for Valheim, 128 for Garry's Mod, 1000 for SA-MP. Where no maximum
is set, the generated tariffs assume 128.

## Ranges: when the customer chooses

The minimum / maximum / step fields turn a tariff into a configurator. A slider
appears when **the type is resource billing** and **the maximum is above the
minimum**. Until both hold, the customer gets exactly what the container limits
say.

- the value from the container limits is the default and the starting point of
  the "from" price;
- minimum and maximum clamp the choice, the step rounds it: with a minimum of 2
  and a step of 2 the customer gets 2, 4 or 6 GB, never 3;
- if only the maximum is filled in, the minimum becomes the container limit.

The "from" price in the list is the cost of the smallest configuration — the
lower bound of every range and the minimum number of slots.

## How many servers fit on a node

A location carries three capacity settings: **server limit**, **memory
overcommit** and **reserved node memory**. Free memory for new servers is:

```
free = node RAM × overcommit − reserve − sum of the limits already handed out
```

The reserve defaults to 1 GB and stays with the node's own system and the agent.
An overcommit of `1.0` hands out memory honestly, `1.5` hands out one and a half
times as much, betting that the servers are not busy at the same time. Memory
overcommit is always a gamble: under simultaneous load the node starts killing
containers.

The location's "Capacity" section shows, per game, how many more servers fit and
what the limit is — memory, disk or the server count. Check it before publishing
a tariff with large resources.

## Periods and discounts

Rental and renewal periods are lists of days, from 1 to 365, and both are
required. The cost of a period follows from the monthly price:

```
period price = monthly price × days ÷ 30 − discount
```

Discounts are an object of "days → percent". For example `{"90": 10, "180": 15}`
gives 10 % for 90 days and 15 % for 180. Anything above 100 is capped, and the
key is always a number of days from the period list.

## Changing the tariff and the resources

A customer may move to another tariff of the same game and location, or change
the resources inside the ranges. The money is settled like this:

- **more expensive** — the difference for the remaining days is charged:
  `(new price − old price) ÷ 30 × days left`, and the expiry date stays;
- **cheaper** — nothing is refunded; the term is extended instead: the remaining
  value is divided by the new daily price and the expiry date moves.

New limits are stored at once but reach the container on the next start: memory,
CPU and the quota are set when the container is created. The panel says as much
— "restart required".

Servers billed through WHMCS are not changed from the panel: renewal, tariff and
resource changes happen on the WHMCS side — [whmcs.md](whmcs.md).

A tariff with servers on it cannot be deleted: the panel asks you to move them
or to simply disable the tariff.

## Tariffs generated from the catalog

When the game catalog is synced, the panel creates three tariffs per game —
"Start", "Optimal" and "Maximum" — derived from the game requirements:

| Tier | RAM | Disk | Cores | Slots |
|---|---|---|---|---|
| Start | game minimum | minimum disk | minimum cores | a quarter of the maximum |
| Optimal | recommended | ×1.5 | +1 core | half of the maximum |
| Maximum | recommended ×1.5 | ×2 | +2 cores | the game maximum |

The price follows `50 + 90 × GB RAM + 120 × core + 3 × GB disk`, rounded up to
ten roubles. Treat it as a sane starting point, not as market advice: put in
your own rates and recalculate.

These tariffs are marked in the database as generated from the catalog and are
refreshed on the next sync — price, cores, memory, disk and the slot maximum go
back to the computed values. If you need your own, duplicate one and edit the
copy, or change only what is not overwritten: name, periods, discounts, ranges.

## Common mistakes

**A range with a zero per-resource price.** The customer drags the slider, the
resources grow, the bill does not move. Fill in the price per core, per GB of
RAM and per GB of disk, or drop the ranges.

**Memory at the game minimum.** The server dies at start or a few minutes into
the game. The server log just stops — that is the OOM killer.

**Disk without quotas.** One server fills the node and the rest stall. Check the
location diagnostics: the `quota` check has to be green.

**Zero cores without shares.** The form refuses to save: with neither a ceiling
nor a share there would be no CPU limit at all.

**A tariff without a location.** It is offered on every location, weak ones
included. Pin a heavy tariff to the location that can carry it.

**Editing a tariff "for everyone".** Servers already handed over keep their
limits. To raise resources for existing customers, move them to the new tariff —
otherwise only new orders see the change.

**Slots in a resource tariff.** Slot bounds do not reach the game settings in
that mode: the player limit is enforced only with slot billing.
