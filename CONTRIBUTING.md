# Contributing to FireMemory

## Requirements

- Go 1.24+
- `make` (or run commands manually)

## Build

```sh
make build
```

Binaries are placed in `./bin/`.

## Test

All tests must pass before submitting a pull request:

```sh
make test
```

Tests run without internet access and without external model dependencies. The `DeterministicEmbedder` is used as a test fixture — do not use real models in tests.

## Lint

```sh
make lint
```

Requires [golangci-lint](https://golangci-lint.run/usage/install/).

## Commit style

Use conventional commits:

```
feat: add auto-download for ONNX models
fix: handle missing brainfile on first run
docs: update quickstart for Cursor
refactor: replace python backend with onnx runtime
test: add reliability test for context engine
```

One logical change per commit. Keep commits small and reviewable.

## Pull request rules

1. All tests must pass (`make test`).
2. No new linter errors (`make lint`).
3. If you add a public function or command, document it.
4. Do not expose `bbolt` internals in the public API — the abstraction is `Brainfile`.
5. Do not add SQL surfaces. See [AGENTS.md](AGENTS.md).
6. FireQuery must not access storage directly. All calls go through FireMemory.
7. New tests must pass offline (no internet, no real ML models).

## Development order

If you are extending the system, follow the mandatory build order in [AGENTS.md](AGENTS.md):

1. FireMemory Core
2. CLI `fmem`
3. Sync / Slow Path
4. Context Engine
5. Basic MCP for FireMemory
6. FireQuery

Do not implement FireQuery features before the relevant FireMemory core is functional.

## Reporting issues

Use the GitHub issue templates. Include:
- OS and architecture
- FireMemory version (`fmem version`)
- Steps to reproduce
- Expected vs actual behavior

## Security issues

Do not open a public issue for security vulnerabilities. See [SECURITY.md](SECURITY.md).
