# Vortanix

Self-hosted control panel for game server hosting. Runs your game servers in
Docker containers across one or many machines, with a web panel for the people
who use them.

## What it does

- **Game servers in containers.** 48 games ship with build recipes in
  [`deploy/images`](deploy/images): install, start, stop, console, files.
- **Many machines, no open ports.** Game nodes connect outward to the panel and
  hold the connection. Nothing has to be exposed on the node itself.
- **A panel for end users.** Console, file manager, backups, scheduled tasks,
  subusers with per-permission access, resource graphs.
- **Backups.** Local and off-site to any S3-compatible storage.
- **Update checks.** An Updates tab asks GitHub whether a newer release exists
  and shows what changed. It does not update itself — that would mean handing
  the panel access to docker on your machine, which is far too much power for
  one button, so it shows you the commands instead. The check runs only when
  you open that tab; clear `UPDATE_REPO` to switch it off entirely.

## Requirements

- A Linux host with Docker and the Compose plugin
- 2 GB of RAM to run the panel, plus whatever the games need
- Another Linux host with Docker for each game node

## Install

On a fresh server, get git and Docker first — `apt install docker.io` will not
do, it ships without the Compose plugin:

```bash
apt-get update && apt-get install -y git curl
curl -fsSL https://get.docker.com | sh
```

Then pull the published images and start:

```bash
git clone https://github.com/vortanixapp/panel
cd panel
sh scripts/init-env.sh
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml up -d
```

`init-env.sh` asks for a domain. Give it one and the panel answers on
`https://` with a Let's Encrypt certificate it obtains by itself. Press Enter
instead and it uses the machine's IP over plain HTTP — you can move to a domain
later by running the same script again with it, and nothing else has to change.

It also generates the secrets. Run it as often as you like: it fills what is
empty, rewrites the addresses, and never touches a secret that is already set.

Open the address the script printed. The setup wizard asks for a panel name and
the owner's email and password — that is the whole setup. There is no licence
key, no activation step, and nothing is sent anywhere: the panel talks to its
own database, its own Redis, and your own game nodes.

Only ports 80 and 443 are published. The panel, the API, the console and the
node link all sit behind them and differ by path, so there is one address to
remember and one certificate to renew.

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

## Security

Do not report vulnerabilities in a public issue — [SECURITY.md](SECURITY.md)
says where they go.

## Licence

MIT — see [LICENSE](LICENSE). Third-party notices are in [NOTICE](NOTICE).
