# Release Checklists

## `v0.1-beta`

Release target:

- internal beta
- controlled technical users
- local-first operation

Status:

- ready to cut

Checklist:

- [x] `go test ./...` passes
- [x] FireMemory CLI commands work on a fresh `.fbrain`
- [x] FireQuery `doctor` works
- [x] FireQuery `devices` works
- [x] backup and restore commands are verified
- [x] integrity validation on open is verified
- [x] format migration from supported legacy version is verified
- [x] JSON diagnostics work in `fmem` and `fquery`
- [x] concurrency and reopen tests pass
- [x] reliability tests pass
- [x] release notes document known limits
- [x] local deploy guide is published
- [x] backup/restore guide is published
- [x] recovery guide is published

Known acceptable limits for beta:

- no formal metrics backend
- no distributed deployment story
- local-first only
- MCP surface intended for controlled integration

## `v1.0`

Release target:

- production use
- supported upgrade path
- operational discipline

Checklist:

- [ ] everything from `v0.1-beta`
- [ ] production hardening tracks are complete
- [ ] compaction behavior is validated under load
- [ ] compatibility policy is frozen
- [ ] MCP schema versioning policy is frozen
- [ ] stable error codes are documented
- [ ] benchmark thresholds are recorded
- [ ] soak tests pass
- [ ] failure injection coverage is documented
- [ ] rollback and recovery procedure is validated
- [ ] signed release process or equivalent release governance is documented
- [ ] support boundaries are documented
