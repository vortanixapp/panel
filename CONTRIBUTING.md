# Contributing

Thanks for taking the time. This is a young public repository — the code is
older than its openness, so expect rough edges in places the closed version
never had to explain.

## Before you start

By taking part you agree to the [Code of Conduct](CODE_OF_CONDUCT.md). It is
short and it says what you would expect.

For anything larger than a bug fix, open an issue first and say what you intend
to do. It costs you a paragraph and can save you a weekend: some areas are
mid-rewrite (see the roadmap in the README) and a patch against them will not
survive.

## Building

```bash
go build ./...          # the panel and the agent
go test ./...           # Go tests
cd web && npm ci && npm run build
```

Everything is one Go module, so `./...` from the repository root does what you
expect.

## Before you open a pull request

```bash
gofmt -l .              # must print nothing
go vet ./...
go test ./...
cd web && npx tsc --noEmit
```

`npm run lint` in `web/` is not wired up yet; the type check is the gate.

## Style

- Comments explain **why**, not what. If a default looks arbitrary, the comment
  should say what broke when it was something else.
- Errors travel up. Do not swallow one into a `nil` or an empty string to make
  a signature tidy — a silent failure here means somebody's game server is
  quietly gone.
- Keep changes narrow. Drive-by renames and reformatting in an otherwise
  unrelated patch make review slow and history hard to read.
- New database work goes in a new migration file. Existing migration files are
  never edited: they have already run on installs you cannot reach.

## Commits and pull requests

Write the commit subject as the result for the user, not the mechanics of the
patch: "Cancelled installs no longer leave a broken server" beats "fix install
bug". No `feat:` / `fix:` prefixes.

One pull request, one topic. If you found a second thing worth fixing, that is
a second pull request — and thank you for finding it.

## Security

Do not open an issue for a vulnerability. [SECURITY.md](SECURITY.md) explains
where to send it.

## Licence

Contributions are accepted under the MIT licence of this repository. There is
no CLA to sign.
