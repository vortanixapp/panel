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

## Quick start

```bash
git clone https://github.com/vortanix/vortanix
cd vortanix
cp deploy/panel/.env.example .env   # edit database credentials
docker compose -f deploy/docker-compose.yml up -d
```

The panel comes up on `http://localhost:3000`. The first visit opens the setup
wizard, which creates the owner account.

Full instructions: [docs/install.md](docs/install.md).

## Layout

| Path | What lives there |
|---|---|
| `cmd/vortanix` | The panel: API, job worker, node relay, console gateway, metrics |
| `cmd/vortanix-agent` | The daemon that runs on each game node |
| `internal/` | Panel implementation |
| `pkg/` | Packages meant to be imported from outside |
| `web/` | Panel front end (Next.js) |
| `migrations/` | Database schema |
| `deploy/` | Compose files, systemd units, game image recipes |

The panel is a single binary. By default it runs every component in one
process, which is what a single-machine install wants. Flags split them apart
when you outgrow that.

## Roadmap

The extraction is not finished. Landing next:

- [ ] Setup without an external licence service
- [ ] Single-tenant mode as the default
- [ ] Prebuilt images on a public registry
- [ ] English as the default interface language

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports go to
[SECURITY.md](SECURITY.md) instead of the issue tracker.

## Licence

MIT — see [LICENSE](LICENSE). Third-party notices are in [NOTICE](NOTICE).
