## What this changes

<!-- The result for the user, in a sentence or two. Not the mechanics. -->

## Why

<!-- What broke, or what was impossible before. Link the issue if there is one. -->

## Checks

- [ ] `gofmt -l .` prints nothing
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] `cd web && npx tsc --noEmit` passes (only if you touched `web/`)

## Notes for the reviewer

<!--
Anything worth knowing: a decision you were unsure about, a case you chose not
to handle, a migration that has to run before this is deployed. If a schema
change is involved, say whether it is safe to apply to a live install.
-->
