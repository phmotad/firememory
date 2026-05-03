# Release Candidate

## Target

Candidate release:

- `v0.1-beta`

Release position:

- ready for internal beta
- not ready for `v1.0`

## Final Cut Checklist

- `go test ./...` passes
- FireMemory acceptance flow passes
- FireQuery acceptance flow passes
- backup and restore are documented
- recovery guide is documented
- compatibility and migration policy is documented
- JSON diagnostics are available for `fmem` and `fquery`
- release notes are written
- known limits are explicitly documented

## Scope of the Beta

Intended use:

- local-first
- controlled environments
- technical users
- integration testing with agents and MCP clients

Not promised in this beta:

- broad production support
- distributed deployment
- full metrics platform
- long-duration soak certification

## Release Decision

Decision:

- cut `v0.1-beta`
- continue hardening toward `v1.0`
