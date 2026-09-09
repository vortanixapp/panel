# Installing Vortanix

This gets you a working panel on one machine. Game nodes are added afterwards,
from inside the panel.

> The panel and the game nodes can be the same machine while you try things
> out. For anything real, keep them apart: a game server that eats all the RAM
> should not take the panel down with it.

## What you need

- A Linux host with Docker and the Compose plugin
- 2 GB of RAM to run the panel, plus whatever the games need
- A domain pointed at the machine, if you want anyone but yourself to use it

Building the images yourself is a different matter: the front end alone wants
**4 GB of RAM**, and on a smaller machine the build does not fail cleanly — it
drives the host into swap until it stops answering, SSH included. The steps
below pull published images instead, which needs no more memory than running
the panel does.

## Preparing the machine

On a fresh server, install git and Docker:

```bash
apt-get update && apt-get install -y git curl
curl -fsSL https://get.docker.com | sh
```

**Do not use `apt install docker.io`**, even though Ubuntu suggests it when
`docker` is missing. That package has no Compose plugin, so `docker compose`
answers `'compose' is not a docker command` and every instruction below fails.
The script above installs the engine, Compose and buildx together.

Check both are there before going on:

```bash
docker --version && docker compose version
```

If you are not root, add yourself to the `docker` group and open a new shell —
otherwise every command needs `sudo`:

```bash
usermod -aG docker "$USER"
```

## Install

```bash
git clone https://github.com/vortanixapp/panel
cd panel
sh scripts/init-env.sh
```

That writes `deploy/.env` and generates four secrets into it. Nothing is printed
to the terminal, so the values do not end up in your shell history or the
server's logs. Run it again whenever you like: it fills what is empty and leaves
alone anything already set, so it cannot wipe the secrets of a working install.

What it generated and why it matters:

- `POSTGRES_PASSWORD` — hex on purpose. It goes into a connection URL, and the
  `/` and `+` of base64 cut that URL in half; the error you get back then talks
  about an invalid port and never mentions the password.
- `JWT_SECRET` — signs sessions. Change it later and everyone is logged out.
- `SECRETS_KEY` — encrypts payment gateway keys and node passwords in the
  database. **If you lose it, the panel cannot decrypt them and you will be
  re-entering every credential by hand.** Keep a copy somewhere you will still
  have it after a disk failure.
- `INTERNAL_SECRET` — authenticates the panel's own services to each other.

### The address, and HTTPS

The script also asks for a domain, and that answer decides how the panel is
reached.

**With a domain** it answers on `https://your-domain` and obtains a Let's
Encrypt certificate on first start, with nothing else to configure. Two things
have to be true before you start, or the certificate cannot be issued: the
domain's A record must point at this machine, and ports 80 and 443 must be
reachable from the internet.

**Without one** it uses the machine's IP over plain HTTP. Everything works, the
traffic is simply not encrypted — fine for trying things out, not for a panel
other people log into.

Moving from an IP to a domain later is one command and a restart:

```bash
sh scripts/init-env.sh panel.example.com
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

Nothing else changes: the panel, the API, the console and the node link all live
on one address and differ by path, so the paths stay the same and only the host
in front of them moves.

Then:

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

That pulls the published images. The database schema is created on the first
start of the API — there is no separate migration step.

To build from source instead, drop the second file:

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

Only do that on a machine with 4 GB of RAM or more, and read the note in
[When it does not come up](#when-it-does-not-come-up) first — a build that runs
out of memory takes the whole host down with it.

Pin a version with `VORTANIX_VERSION` in `deploy/.env` — `latest` moves under
you on the next pull, which is rarely what you want on a machine other people
depend on. Image tags carry no `v`: the git tag is `v0.1.0`, the image tag is
`VORTANIX_VERSION=0.1.0`.

Open the address the script printed. The setup wizard asks for a panel name and
the owner's email and password. That is the whole setup: there is no licence
key, no activation, and nothing is sent anywhere.

## Adding a game node

A node is any Linux machine with Docker that will actually run the game
servers. Add it in **Admin → Locations**, then run the command the panel gives
you on that machine. The node connects outward to the relay and holds the
connection open, so you do not need to open any inbound port on it.

### Game images are yours to build

The panel ships the recipes, not the images: `deploy/images` holds four runtime
images that all 48 games share. Nobody publishes them for you, so build them
once and push them to a registry your nodes can reach:

```bash
VORTANIX_REGISTRY=ghcr.io/your-account sh scripts/build-game-images.sh --runtimes --push
```

Then set `VORTANIX_REGISTRY` to the same value for the panel and the worker, so
they hand nodes an address that resolves. Skip this and creating a server fails
at the pull with `pull access denied`, which is the single most common way a
fresh install looks broken.

Adding a game later needs no rebuild — games differ by catalogue data, not by
image.

## Behind a proxy you already run

The stack brings its own — Caddy on ports 80 and 443, which is where the
certificate comes from. If you already terminate TLS somewhere else, set
`HTTP_PORT` and `HTTPS_PORT` in `deploy/.env` to free ports, point your own
proxy at them, and keep the four paths intact:

| Path | Goes to |
|---|---|
| `/v1/agent/*` | game nodes, a WebSocket |
| `/v1/console`, `/v1/dashboard` | the server console, a WebSocket |
| `/v1/*` | the API |
| everything else | the web app |

A proxy that does not upgrade connections gives you a panel where everything
works except the console and the nodes — both are WebSockets.

Then run `sh scripts/init-env.sh your-domain` so the addresses handed to nodes
and put in emails match what people actually type.

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
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml pull
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

`git pull` alone changes nothing that is running: the containers come from
published images, and only `deploy/.env` and `deploy/Caddyfile` are read from
the checkout. The `pull` line is what actually brings new code down.

If you changed `deploy/Caddyfile`, restart Caddy explicitly — it reads that file
once at start, and Compose will not recreate a container just because a mounted
file changed:

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d --force-recreate caddy
```

New migrations run on start. Take a database backup first — the migrations only
move forward, and there is no downgrade path.

Every push to `main` publishes images tagged `edge`; every release tag publishes
`latest` and the version. `VORTANIX_VERSION` in `deploy/.env` picks which you
follow — `edge` to track development, a version to stay put.

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
- **You changed `POSTGRES_PASSWORD` and now the API cannot connect** — Postgres
  sets that password once, when it first creates its data directory. Changing
  the variable afterwards changes what the panel sends, not what the database
  expects. Change it inside the database instead (`ALTER USER vortanix WITH
  PASSWORD '…'`), or wipe the volume and start over.
- **The machine stopped answering during `--build`, SSH included** — the front
  end build ran out of memory. It does not fail with an error: the host swaps
  until nothing responds. Note that killing your SSH session does not stop it,
  because the build runs inside the Docker daemon, not in your shell — reboot
  the machine from your provider's console, then use the published images
  instead of building. If you must build on a small machine, add swap first:

  ```bash
  fallocate -l 4G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
  ```
