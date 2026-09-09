# Installing and operating Vortanix

[English](install.md) · [Русский](install.ru.md)

This document covers the full lifecycle: server preparation, installation,
initial configuration, adding game nodes, backups, upgrades, rollback and
diagnostics.

For an evaluation install, the commands in the [README](../README.md) are
sufficient. This document is intended for production use.

- [Planning](#planning)
- [Preparing the machine](#preparing-the-machine)
- [Installation](#installation)
- [The deploy/.env file and secrets](#the-deployenv-file-and-secrets)
- [Panel address and HTTPS](#panel-address-and-https)
- [First start](#first-start)
- [Mail](#mail)
- [Game images](#game-images)
- [Adding a game node](#adding-a-game-node)
- [Backups](#backups)
- [Versions and update channels](#versions-and-update-channels)
- [Upgrading](#upgrading)
- [Rollback](#rollback)
- [Operating behind an existing reverse proxy](#operating-behind-an-existing-reverse-proxy)
- [Operations](#operations)
- [Diagnostics](#diagnostics)
- [Removal](#removal)

## Planning

**Composition.** The panel machine runs the panel, the database and the channel
to the nodes. A game node runs the game servers. Combining both roles on one
machine is acceptable for evaluation. In production the roles should be
separated: resource exhaustion by a game server must not cause the panel to
fail.

| Machine | Minimum | Recommended |
|---|---|---|
| Panel | 2 GB RAM, 2 cores, 20 GB disk | 4 GB RAM, 4 cores, 40 GB SSD |
| Game node | as required by the games | one machine per high-load game |

Building the panel images from source requires 4 GB of RAM regardless of the
figures above. This document describes installation from published images, for
which no additional memory is required.

**Architecture.** The published images are built for x86-64 (amd64) only, both
for the panel and for the agent. On other architectures — including arm64 — the
images have to be built from source on the target machine.

**Domain name.** The domain should be prepared before the first start: the
certificate is issued at that moment. Two conditions must hold:

- an A record for the domain points to the panel machine;
- ports 80 and 443 on that machine are reachable from the internet.

Without a domain the panel operates over the machine's IP address and plain
HTTP. That mode is acceptable for evaluation only: user credentials are
transmitted in clear text.

**Network access.** The panel machine requires inbound ports 80 and 443. A game
node requires no inbound ports for the panel — the agent establishes an
outbound connection. Only the ports used by the game servers themselves need to
be opened.

## Preparing the machine

```bash
apt-get update && apt-get install -y git curl
```

```bash
curl -fsSL https://get.docker.com | sh
```

The `docker.io` package should not be used, although Ubuntu suggests it when
the `docker` command is missing. That package contains no Compose plugin:
`docker compose` reports `'compose' is not a docker command` and the subsequent
steps cannot be performed. The script above installs the engine, Compose and
buildx together.

Verify that both components are present:

```bash
docker --version && docker compose version
```

When not operating as root, add the user to the `docker` group and open a new
session; otherwise every command requires `sudo`:

```bash
usermod -aG docker "$USER"
```

## Installation

```bash
git clone https://github.com/vortanixapp/panel
cd panel
sh scripts/init-env.sh
```

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

Eight containers are started: PostgreSQL, Redis, the API, the job worker, the
relay, the console, the web interface and Caddy. Only Caddy publishes ports.

The database schema is created on the first start of the API. A separate
migration step is not required.

Container status:

```bash
docker compose -f deploy/docker-compose.yml ps
```

## The deploy/.env file and secrets

`init-env.sh` creates `deploy/.env`, generates four secrets in it and sets the
addresses. Values are not written to the terminal and therefore do not appear
in shell history or server logs. Re-running the script is safe: it fills empty
values and rewrites addresses without modifying secrets that are already set.

| Variable | Purpose | Consequence of loss |
|---|---|---|
| `POSTGRES_PASSWORD` | database access | see the note below |
| `JWT_SECRET` | session signing | all sessions are terminated |
| `SECRETS_KEY` | encryption of payment gateway keys and node credentials | the data cannot be decrypted |
| `INTERNAL_SECRET` | mutual authentication between panel services | services stop trusting each other |

`POSTGRES_PASSWORD` is generated in hexadecimal form. The password is
substituted into the connection string; the characters `/` and `+`, which occur
in base64, break its parsing, and the resulting error refers to an invalid port
without mentioning the password.

`SECRETS_KEY` must be stored separately from the server. Payment gateway keys
and node credentials are encrypted with it. A database backup without this key
is incomplete: the encrypted columns cannot be recovered.

The remaining parameters are documented in
[`deploy/.env.example`](../deploy/.env.example): mail, external sign-in
providers, the game image registry, CORS and the update check.

## Panel address and HTTPS

The answer to the script's domain prompt determines the operating mode.

**With a domain**, the panel responds at `https://domain`. A Let's Encrypt
certificate is issued on first start with no further configuration, and renewal
is automatic. If no certificate is issued, the cause is almost always one of
the two conditions stated above: the A record, or the availability of port 80.

**Without a domain**, the machine's IP address and plain HTTP are used.

Migration from an IP address to a domain requires two commands:

```bash
sh scripts/init-env.sh panel.example.com
```

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

No other changes are required. The panel, the API, the console and the node
channel are served from one address and are distinguished by request path, so
the paths remain unchanged and only the host name differs. Connected game nodes
receive the new relay address at their next reconnection.

## First start

Open the address printed by the script. The setup wizard requests a panel name
and the owner's email address and password. No licence key or activation is
required, and no data is transmitted to external services.

The owner account created here is the only account with access to the
administration area until such access is granted to other users.

A message stating that the panel is already configured indicates that an owner
account exists. In that case, sign in instead.

## Mail

Without SMTP configured the panel operates, but registration confirmations,
password resets and payment notifications are not sent. The parameters are set
in `deploy/.env` and require a restart:

```
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
MAIL_FROM=
```

`MAIL_FROM` must be an address the SMTP server permits sending from. Otherwise
messages are accepted for delivery and then discarded by the receiving side
without notification.

Links in these messages are built from `FRONTEND_URL`, which is set by
`init-env.sh`. If messages contain an incorrect address, re-run the script.

## Game images

The repository contains build recipes, not prebuilt game images. The
`deploy/images` directory holds four runtime base images shared by all 47
games. These images are not published.

Without the steps below, creating a game server fails with `pull access denied`
at the image download stage.

Build the images and publish them to a registry reachable from the game nodes:

```bash
VORTANIX_REGISTRY=ghcr.io/your-account sh scripts/build-game-images.sh --runtimes --push
```

Then set the same registry in `deploy/.env` so that the panel and the job
worker pass a resolvable address to the nodes:

```
VORTANIX_REGISTRY=ghcr.io/your-account
```

Adding a game later does not require a rebuild: games differ by catalogue data,
not by image. The full set of script options is printed by
`sh scripts/build-game-images.sh --help`.

For a private registry, each node requires `docker login` to be run once.

## Adding a game node

Any machine running Linux with Docker can serve as a game node. Preparation is
identical to the panel machine, as described in
[Preparing the machine](#preparing-the-machine).

In the panel, open **Administration → Locations** and create a location. The
panel displays a ready command containing the relay address, the node
identifier and its token. Run it on the node. The agent establishes an outbound
connection to the relay and holds it; no inbound ports need to be opened on the
node.

The node reports as online within a few seconds. If no connection is
established:

```bash
docker logs vortanix-agent --tail 50
```

The most common causes are a relay address that the node cannot reach or
resolve, and a token that was regenerated in the panel after the command was
copied.

If the panel declines to produce the command and reports that
`RELAY_PUBLIC_URL` is not set, `deploy/.env` contains no external relay
address. Run `sh scripts/init-env.sh` again and restart the panel.

To deploy nodes from a configuration management system, the
[`deploy/agent`](../deploy/agent) directory contains an equivalent Compose file
and script. The values `RELAY_URL`, `AGENT_TOKEN` and `NODE_ID` are taken from
the panel.

## Backups

A backup consists of three independent parts. Any one of them alone is not
sufficient for recovery.

**The database.**

```bash
docker compose -f deploy/docker-compose.yml exec -T postgres pg_dump -U vortanix vortanix > vortanix-$(date +%F).sql
```

**The `SECRETS_KEY` value** from `deploy/.env`, stored separately. Without it
the encrypted columns of the dump cannot be recovered.

**Uploaded files** — branding images and catalogue archives, in the
`vortanix_uploads` volume.

```bash
docker run --rm -v vortanix_uploads:/from -v "$PWD":/to alpine tar czf /to/uploads-$(date +%F).tar.gz -C /from .
```

Game server files reside on the nodes and are not included in the above. They
are backed up by the panel's own facilities, optionally to S3-compatible
off-site storage. Storage is configured in **Administration → Settings →
Storage**; its credentials are among the values encrypted with `SECRETS_KEY`.

Restoring the database:

```bash
docker compose -f deploy/docker-compose.yml exec -T postgres psql -U vortanix -d vortanix < vortanix-2026-01-01.sql
```

## Versions and update channels

The `VORTANIX_VERSION` variable in `deploy/.env` determines which images the
overlay file pulls.

| Value | Result |
|---|---|
| unset or `latest` | the most recent release; changes on every pull |
| `0.1.1` | that release and no other |
| `edge` | the current state of `main`, rebuilt on every push |

On machines in production use, the version should be pinned explicitly.

Image tags carry no `v` prefix: repository tag `v0.1.1` corresponds to
`VORTANIX_VERSION=0.1.1`.

The Updates section of the administration area reports the availability of a
new release and its changes. It does not install updates; the procedure is
given below.

## Upgrading

Create a database backup before upgrading. Migrations are applied in the
forward direction only; no schema downgrade procedure exists, and reversion is
performed by restoring from a backup.

```bash
cd panel && git pull
```

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml pull
```

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

`git pull` does not alter the state of running containers. Containers are
created from published images; only `deploy/.env` and `deploy/Caddyfile` are
read from the working copy. New code is delivered by the `pull` command.
Omitting it is the most common reason an upgrade produces no effect.

New migrations are applied when the API starts. For several seconds the panel
responds slowly or not at all.

If the upgrade modified `deploy/Caddyfile`, restart Caddy explicitly. The file
is read once at start, and Compose does not recreate a container because a
mounted file changed:

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d --force-recreate caddy
```

Game nodes are upgraded separately and not necessarily at the same time as the
panel: agents remain operational across versions. Re-running the command from
**Administration → Locations** on a node pulls the current agent image and
replaces the container.

## Rollback

If a release is defective, set the previous version in `deploy/.env`:

```
VORTANIX_VERSION=0.1.0
```

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

This procedure applies only when the release being left applied no migrations.
Otherwise the earlier code addresses a schema unknown to it, and the correct
procedure is:

1. stop the stack with `docker compose … down`;
2. restore the database dump taken before the upgrade;
3. set the earlier `VORTANIX_VERSION` and start the stack.

For this reason the backup precedes the upgrade.

## Operating behind an existing reverse proxy

The panel includes Caddy, which occupies ports 80 and 443 and performs
certificate issuance. If TLS is terminated by another server, set free values
for `HTTP_PORT` and `HTTPS_PORT` in `deploy/.env`, direct the existing proxy to
them, and preserve the path routing:

| Path | Destination |
|---|---|
| `/v1/agent/*` | game nodes, WebSocket |
| `/v1/console`, `/v1/dashboard` | server console, WebSocket |
| `/v1/*` | API |
| everything else | web interface |

A proxy without WebSocket support results in a failure of the console and of
node connectivity while the remaining functions continue to operate.

After configuration, run `sh scripts/init-env.sh domain` so that the addresses
supplied to nodes and included in mail match the address in actual use.

## Operations

All commands require both `-f` parameters, so defining an alias is advisable:

```bash
alias vx='docker compose -f ~/panel/deploy/docker-compose.yml -f ~/panel/deploy/docker-compose.images.yml'
```

| Task | Command |
|---|---|
| Container status | `vx ps` |
| API log in real time | `vx logs -f api` |
| Restart a single service | `vx restart api` |
| Stop | `vx down` |
| Start | `vx up -d` |
| Database console | `vx exec postgres psql -U vortanix vortanix` |
| Disk usage | `docker system df -v` |

Once initialised, the API responds to `GET /health`; this endpoint is suitable
for external monitoring.

## Diagnostics

```bash
docker compose -f deploy/docker-compose.yml ps
```

```bash
docker compose -f deploy/docker-compose.yml logs api
```

- **The `api` container restarts repeatedly.** The cause is stated in its log.
  Typically this is an unset `POSTGRES_PASSWORD` or an unreachable database.
- **The panel loads but every request fails.** The browser is addressing an API
  at an address that is not served. Run `sh scripts/init-env.sh` with the
  address in actual use and verify that it is listed in `CORS_ORIGINS`.
- **The setup wizard reports that the panel is already configured.** An owner
  account exists. Sign in, or delete the database volume and start again with
  `docker compose down -v`, which destroys all data.
- **No certificate is issued.** The A record does not point to this machine, or
  port 80 is closed: Let's Encrypt validation uses port 80 even for an HTTPS
  certificate. The specific cause is stated in `docker compose logs caddy`.
- **After changing `POSTGRES_PASSWORD` the API cannot connect.** PostgreSQL
  sets the password once, when it creates its data directory. Changing the
  variable changes the password sent by the panel, not the one expected by the
  database. Change it inside the database with `ALTER USER vortanix WITH
  PASSWORD '…'`, or delete the volume and install again.
- **Creating a server fails with `pull access denied`.** The game images have
  not been built and published; see [Game images](#game-images).
- **The machine stopped responding during a `--build` run, SSH included.** The
  web interface build exhausted available memory. No error is produced: the
  system begins heavy swapping and stops responding. Terminating the SSH
  session does not stop the build, which runs inside the Docker daemon. Reboot
  the machine from the hosting provider's control panel and use the published
  images. If building on a machine with limited memory is required, add swap
  space first:

  ```bash
  fallocate -l 4G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
  ```

## Removal

Stopping with data retained:

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml down
```

Stopping with the database, Redis, uploaded files and the certificate deleted.
The operation is irreversible:

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml down -v
```

On each game node:

```bash
docker rm -f vortanix-agent
```

Game server files remain in `/var/lib/vortanix/servers` until deleted manually.
