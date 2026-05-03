## What this PR does

<!-- One paragraph. What changed and why. -->

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Refactor
- [ ] Documentation
- [ ] Test
- [ ] Build / CI

## Checklist

- [ ] `go test ./...` passes
- [ ] No new linter errors (`golangci-lint run ./...`)
- [ ] New tests are offline-safe (no internet, no real ML models)
- [ ] Public API does not expose `bbolt` internals
- [ ] No SQL surface introduced
- [ ] If FireQuery was changed: FireQuery does not access storage directly

## Related issues

<!-- Closes #... -->
