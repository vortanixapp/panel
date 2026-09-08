# Vortanix

Self-hosted control panel for game server hosting. Runs your game servers in
Docker containers across one or many machines, with a web panel for the people
who use them.

> **Status: pre-release.** The code is being extracted from a closed-source
> product. It builds and its tests pass, but the single-tenant install path is
> still being finished — see [Roadmap](#roadmap). Do not run it in production
> yet.

## What it does

- **Game servers in containers.** 48 games ship with build recipes in
  [`deploy/images`](deploy/images): install, start, stop, console, files.
- **Many machines, no open ports.** Game nodes connect outward to the panel and
  hold the connection. Nothing has to be exposed on the node itself.
- **A panel for end users.** Console, file manager, backups, scheduled tasks,
  subusers with per-permission access, resource graphs.
- **Backups.** Local and off-site to any S3-compatible storage.

## Requirements

- Linux host with Docker for the game nodes
- PostgreSQL 16+ and Redis for the panel
- Go 1.26 and Node 22+ to build from source

## Install

```bash
git clone https://github.com/vortanix/vortanix
cd vortanix
cp deploy/.env.example deploy/.env   # fill in the three secrets it asks for
docker compose -f deploy/docker-compose.yml up -d --build
```

Open `http://localhost:3000`. The setup wizard asks for a panel name and the
owner's email and password — that is the whole setup. There is no licence key,
no activation step, and nothing is sent anywhere: the panel talks to its own
database, its own Redis, and your own game nodes.

The database schema is created on first start; there is no separate migration
command.

Step by step, including reverse proxies, backups and what to do when it does
not come up: [docs/install.md](docs/install.md).

That builds the images from source. To pull published ones instead, add the
overlay:

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

## Building without Docker

```bash
go build ./...
cd web && npm ci && npm run build
```

You then need a PostgreSQL and a Redis of your own, and the environment
variables listed in `deploy/.env.example`.

## Layout

| Path | What lives there |
|---|---|
| `cmd/vortanix-api` | The panel API |
| `cmd/vortanix-worker` | Job queue: provisioning, backups, mail |
| `cmd/vortanix-relay`, `cmd/vortanix-console` | The link to game nodes |
| `cmd/vortanix-metrics`, `cmd/vortanix-status` | Metrics intake, status page |
| `cmd/vortanix-agent` | The daemon that runs on each game node |
| `internal/` | Panel implementation |
| `pkg/` | Packages meant to be imported from outside |
| `web/` | Panel front end (Next.js) |
| `migrations/` | Database schema |
| `deploy/` | Compose files, systemd units, game image recipes |

One panel, one owner, one database. There is no multi-tenancy and no tenant
registry: this is software you run for yourself, not a platform hosting other
people's panels.

## Roadmap

The extraction is not finished. Landing next:

- [x] Setup without an external licence service
- [x] One panel, one owner, one database
- [x] A compose file and an install guide
- [x] Images published to GHCR on every tag
- [x] English as the default interface language
- [x] A landing page that ships no artwork we do not own
- [ ] API error messages in English (they are still Russian)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports go to
[SECURITY.md](SECURITY.md) instead of the issue tracker.

## Licence

MIT — see [LICENSE](LICENSE). Third-party notices are in [NOTICE](NOTICE).
