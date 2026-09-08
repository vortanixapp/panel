# Development

## Checks

```bash
go build ./...                    # from the root: this is one module
go vet ./...
go test ./...
gofmt -l .                        # must print nothing
cd web && npx tsc --noEmit
```

`npm run lint` in `web/` is a trap: eslint is not set up, and `next lint` opens
an interactive wizard and hangs. The type check is the gate.

## Layout

| Path | What lives there |
|---|---|
| `cmd/vortanix-*` | Entry points: api, worker, relay, console, metrics, status |
| `cmd/vortanix-agent` | The daemon that runs on each game node |
| `internal/api` | Panel: auth, servers, nodes, payments taken from end users |
| `internal/agent` | Docker, files and console on the node |
| `internal/relay`, `internal/console` | The link to nodes — nodes open no inbound ports |
| `internal/worker` | Job queue |
| `pkg/` | Shared code: gamecatalog, protocol, secretbox |
| `web/` | Front end, Next.js + shadcn/ui |
| `migrations/core` | Schema. `deploy/images` holds build recipes for 48 games |

## Style

- **No comments.** The code carries its meaning in names and structure. If a
  line needs a sentence beside it to be understood, rewrite the line.
- Build and tool directives are not comments and must stay: `//go:build`,
  `//go:generate`, `//nolint`, `// Deprecated:`, `// @ts-ignore`,
  `/* eslint-disable */`. Removing one breaks the build or the checks.
- The reasoning goes in the commit message, and that makes the commit message
  load-bearing: say what was broken and why this answer, not what the patch
  touched. No `feat:` / `fix:` prefixes — the subject is the result for the
  user.
- Errors travel up. Do not swallow one into `nil` or `?? ""` to tidy a
  signature: a silent failure here means somebody's game server is quietly
  gone.

## Schema and migrations

- One install, one owner, one database. The `tenant_id` column is left over
  from the multi-tenant version and holds a constant: it appears in hundreds of
  queries, and taking it out for tidiness would cost more than it is worth.
- A new migration takes the next free number and only moves forward. Number
  `055` is skipped — do not reuse it.
- Existing migration files are never edited: the same directory is applied to
  installs you cannot reach.

## Database access

`h.dbOf(ctx)` and `h.readerOf(ctx)` both return the single database. The names
and signatures survive from the multi-tenant version, where they chose between
pools; there are around 700 call sites and rewriting them would gain nothing.
