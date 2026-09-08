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

## Building

```bash
git clone https://github.com/vortanix/vortanix
cd vortanix
go build ./...
cd web && npm ci && npm run build
```

Point it at a PostgreSQL database and a Redis, run the migrations in
`migrations/core`, start `vortanix-api` and the web app, and the first visit
opens the setup wizard: panel name, owner email, owner password. Nothing else
is asked for, and nothing is checked against any server of ours — the panel has
no licence and no phone-home.

**There is still no packaged install.** The compose files under `deploy/` are
the ones the closed product used: they pull prebuilt images from a private
registry and will not work for you. Replacing them, and publishing images
anyone can pull, is the next thing on the roadmap.

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
- [ ] A compose file and an install guide that work from a clean machine
- [ ] Prebuilt images on a public registry
- [ ] English as the default interface language
- [ ] Landing page artwork we hold the rights to

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports go to
[SECURITY.md](SECURITY.md) instead of the issue tracker.

## Licence

MIT — see [LICENSE](LICENSE). Third-party notices are in [NOTICE](NOTICE).
