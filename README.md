# Vortanix

[English](README.md) · [Русский](README.ru.md)

A self-hosted control panel for game server hosting. Game servers run in Docker
containers across one or more machines. End users get a client area;
administrators get management of nodes, tariffs and billing.

The panel is designed for a single owner and a single database. Multi-tenancy,
licence keys and activation are not part of it.

## Features

- **Game servers in containers.** 47 games with build recipes in
  [`deploy/images`](deploy/images): installation, start, stop, console and file
  access.
- **Distribution across machines.** Game nodes open an outbound connection to
  the panel and hold it. No inbound port has to be opened on a node.
- **Client area.** Console, file manager, backups, scheduled tasks, subusers
  with individual permissions, resource graphs.
- **Billing.** Tariffs, account balance, promotions, payment gateways,
  documents.
- **Backups.** Local, and off-site to any S3-compatible storage.
- **Update checks.** The Updates section queries GitHub for new releases and
  displays the changes. Installation is performed manually by the
  administrator: the panel has no access to Docker on the host. The request is
  made only when the section is opened. An empty `UPDATE_REPO` disables the
  check.

## Requirements

| Machine | Minimum | Recommended |
|---|---|---|
| Panel | 2 GB RAM, 2 cores, 20 GB disk | 4 GB RAM, 4 cores, 40 GB SSD |
| Game node | as required by the games | one machine per high-load game |

A Linux distribution with Docker and the Compose plugin is required. Building
the images from source requires an additional 4 GB of RAM; using the published
images, as described below, requires no additional memory.

The published images are built for x86-64 (amd64) only. On other architectures
the images have to be built from source.

The panel and the game servers should be placed on separate machines. Resource
exhaustion by a game server must not cause the panel to fail.

## Installation

Install git and Docker. The `docker.io` package is not suitable: it ships
without the Compose plugin, and the commands below then fail with
`'compose' is not a docker command`.

```bash
apt-get update && apt-get install -y git curl
curl -fsSL https://get.docker.com | sh
```

Deploy the panel:

```bash
git clone https://github.com/vortanixapp/panel
cd panel
sh scripts/init-env.sh
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

`init-env.sh` generates four secrets and requests a domain name. If a domain is
supplied, the panel is served over HTTPS with a Let's Encrypt certificate
issued automatically. If no domain is supplied, the machine's IP address and
plain HTTP are used. A domain can be set later by re-running the script; no
other changes are required.

Open the address printed by the script. The setup wizard requests a panel name
and the owner's email address and password. Setup is then complete. No data is
transmitted to external services: the panel communicates only with its own
database, its Redis instance and the game nodes.

The database schema is created on first start. A separate migration command is
not required.

Only ports 80 and 443 are published. The panel, the API, the console and the
node channel sit behind them and are distinguished by request path.

**Full documentation** — hardware requirements, domain and mail configuration,
building game images, adding nodes, backups, upgrades, rollback, operation
behind an existing reverse proxy, and diagnostics:
[docs/install.md](docs/install.md).

## Building from source

To build the panel images instead of pulling them, omit the overlay file. This
requires 4 GB of RAM; see the
[note on memory exhaustion](docs/install.md#diagnostics) first.

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

Building without Docker:

```bash
go build ./...
cd web && npm ci && npm run build
```

Separate PostgreSQL and Redis instances are then required, along with the
environment variables listed in [`deploy/.env.example`](deploy/.env.example).

## Repository layout

| Path | Contents |
|---|---|
| `cmd/vortanix-api` | Panel API |
| `cmd/vortanix-worker` | Job queue: provisioning, backups, mail |
| `cmd/vortanix-relay`, `cmd/vortanix-console` | Channel to game nodes |
| `cmd/vortanix-metrics`, `cmd/vortanix-status` | Metrics intake, status page |
| `cmd/vortanix-agent` | Service installed on each game node |
| `internal/` | Panel implementation |
| `pkg/` | Packages intended for external use |
| `web/` | Web interface (Next.js) |
| `migrations/` | Database schema |
| `deploy/` | Compose files, agent deployment, game image recipes |

## Security

Report vulnerabilities as described in [SECURITY.md](SECURITY.md). Disclosure
in a public issue is not permitted.

## Licence

MIT; the text is in [LICENSE](LICENSE). Third-party notices are in
[NOTICE](NOTICE).
