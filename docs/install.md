# Installing Vortanix

This gets you a working panel on one machine. Game nodes are added afterwards,
from inside the panel.

> The panel and the game nodes can be the same machine while you try things
> out. For anything real, keep them apart: a game server that eats all the RAM
> should not take the panel down with it.

## What you need

- A Linux host with Docker and the Compose plugin
- 2 GB of RAM for the panel itself, plus whatever the games need
- A domain pointed at the machine, if you want anyone but yourself to use it

## Install

```bash
git clone https://github.com/vortanix/vortanix
cd vortanix
cp deploy/.env.example deploy/.env
```

Open `deploy/.env` and fill in the three secrets. Generate each one separately —
do not reuse the same value:

```bash
openssl rand -base64 32
```

`POSTGRES_PASSWORD` is the database password. `JWT_SECRET` signs sessions:
change it later and everyone is logged out. `SECRETS_KEY` encrypts payment
gateway keys and node passwords in the database — **if you lose it, the panel
cannot decrypt them and you will be re-entering every credential by hand.**
Keep a copy somewhere you will still have it after a disk failure.

Set `PANEL_URL` to the address the browser will use. `http://localhost:8080` is
fine while you are on the machine itself; for anyone else it has to be the real
address, or the panel will load and every request from it will fail.

Then:

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

The first build takes a few minutes. The database schema is created on the
first start of the API — there is no separate migration step.

Open `http://localhost:3000`. The setup wizard asks for a panel name and the
owner's email and password. That is the whole setup: there is no licence key,
no activation, and nothing is sent anywhere.

## Adding a game node

A node is any Linux machine with Docker that will actually run the game
servers. Add it in **Admin → Locations**, then run the command the panel gives
you on that machine. The node connects outward to the relay and holds the
connection open, so you do not need to open any inbound port on it.

## Behind a reverse proxy

Terminate TLS in front of the panel and pass two things through:

- `/` to the web container (port 3000)
- the API to the api container (port 8080)

Point `PANEL_URL` and `CORS_ORIGINS` at the public HTTPS address. The console
uses a WebSocket on port 8083 — a proxy that does not upgrade connections will
give you a panel where everything works except the server console.

## Backups

Two things matter and they are not the same:

- **The database** — everything the panel knows. `docker compose exec postgres
  pg_dump -U vortanix vortanix > backup.sql`
- **`SECRETS_KEY`** — without it a database backup is only half a backup: the
  encrypted columns stay encrypted.

Game server files live on the nodes, not here. Back those up from the panel's
own backup feature, or from the node.

## Upgrading

```bash
git pull
docker compose -f deploy/docker-compose.yml up -d --build
```

New migrations run on start. Take a database backup first — the migrations only
move forward, and there is no downgrade path.

## When it does not come up

```bash
docker compose -f deploy/docker-compose.yml ps
docker compose -f deploy/docker-compose.yml logs api
```

- **`api` restarts in a loop** — read its log. A missing `POSTGRES_PASSWORD` or
  an unreachable database is the usual cause, and it says so.
- **The panel loads but every request fails** — `PANEL_URL` does not match the
  address in the browser's address bar, so the browser is calling an API that
  is not there. `CORS_ORIGINS` has to list that address too.
- **The setup wizard says the panel is already configured** — it is: an owner
  exists. Log in instead, or drop the database volume and start over
  (`docker compose down -v`, which deletes everything).
