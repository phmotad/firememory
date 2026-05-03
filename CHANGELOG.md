# Changelog

## `v0.1-beta` - 2026-05-02

Release type:

- internal beta
- local-first
- controlled technical usage

Highlights:

- FireMemory core implemented
- `.fbrain` Brainfile with bbolt hidden behind public abstractions
- `remember`, `recall`, `sync`, `get_context`, and `explain`
- CLI `fmem`
- backup, restore, integrity validation, compaction, and compatibility migration
- FireQuery implemented as the cognitive interface layer
- strict external and internal contracts
- specialist pipeline with CPU fallback
- CLI `fquery`
- MCP tool surface:
  - `firequery.ask`
  - `firequery.plan`
  - `firequery.remember`
  - `firequery.recall`
  - `firequery.get_context`
  - `firequery.explain`
- structured JSON diagnostics
- reliability and compatibility test coverage

Operational status:

- ready for internal beta
- not yet declared `v1.0`

Known limits:

- no formal metrics backend
- no distributed deployment model
- no automatic downgrade migration
- no soak-test-based production claim yet
