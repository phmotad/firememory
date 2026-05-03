# Production Hardening

## Status

FireMemory + FireQuery are functionally complete at the MVP level.

This does not yet mean production readiness.

Production readiness requires operational hardening, failure handling, compatibility guarantees, and release discipline.

## Release Targets

## `v0.1-beta`

Target:

- safe internal beta
- controlled workloads
- technical users
- local-first operation

Required before beta:

- structured logging for CLI, MCP, and pipeline boundaries
- stable error codes for FireMemory and FireQuery
- explicit config surface for runtime and thresholds
- backup and restore commands for `.fbrain`
- migration policy for `format_version`
- corruption detection on open
- concurrency tests for multi-open and repeated write/read cycles
- file-size and large-memory regression tests
- documented resource limits
- beta deployment guide

## `v1.0`

Target:

- general production usage
- versioned upgrade path
- operational support

Required before `v1.0`:

- compatibility policy for `.fbrain`
- real compaction strategy
- recovery workflow for damaged brainfiles
- full observability story
- metrics and health endpoints or equivalent machine-readable diagnostics
- audited MCP contract stability
- benchmark suite with budget thresholds
- soak tests
- failure injection tests
- release checklist and signed versioning policy

## Hardening Tracks

## 1. Storage Safety

- implement real `Compact`
- add backup and restore support
- add integrity checks for manifest and namespaces
- define recovery behavior for partial writes
- define lock strategy for concurrent access

## 2. Compatibility

- define migration contract for `.fbrain`
- add format compatibility tests across versions
- define unsupported-version behavior
- document upgrade and downgrade policy

## 3. Observability

- add structured logs
- define stable error codes
- emit machine-readable diagnostics
- expose trace boundaries consistently

## 4. Reliability

- add repeated open/close tests
- add concurrent read/write tests
- add large-file regression tests
- add failure injection around storage and embedding boundaries

## 5. Runtime Safety

- centralize config
- define memory budgets and defaults
- validate path handling and sandbox expectations
- harden FireQuery startup checks

## 6. MCP Surface

- define versioning rules for tools and schemas
- add compatibility tests for MCP payloads
- add examples for rejection flows
- define deprecation policy

## Recommended Order

1. storage safety
2. compatibility and migrations
3. observability
4. reliability and soak tests
5. MCP contract stabilization
6. beta release checklist

## Exit Criteria

The project should only be called production-ready when:

- `go test ./...` passes consistently
- backup and restore are verified
- corruption handling is defined and tested
- format upgrades are tested
- runtime diagnostics are stable
- core operations are benchmarked under expected load
- operational docs exist for install, upgrade, backup, restore, and recovery

## Operational Docs

- local deploy guide: `docs/operations/deploy-local.md`
- backup and restore guide: `docs/operations/backup-restore.md`
- recovery guide: `docs/operations/recovery.md`
- release checklists: `docs/release/release-checklists.md`
