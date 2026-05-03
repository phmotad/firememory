# AGENTS.md

FireMemory is a local-first Brainfile for AI agents.

This is the primary repository instruction file for any AI agent working in this codebase.

## Scope

FireMemory stores operational memory in a single `.fbrain` file.

Core cognitive operations:

- `remember`
- `recall`
- `relate`
- `forget`
- `consolidate`
- `get_context`
- `explain`
- `sync`

FireQuery is the cognitive interface layer over FireMemory. It must respect strict contracts and must not bypass FireMemory.

## Canonical Development Order

The build order is mandatory:

1. FireMemory Core
2. CLI `fmem`
3. Sync / Slow Path
4. Context Engine
5. Basic MCP for FireMemory
6. FireQuery

Do not implement FireQuery before FireMemory is functional.

## Non-Negotiable Rules

### 1. No SQL surface

Do not create:

- SQL parsers
- SQL-like languages
- SQL query abstractions

FireMemory is a cognitive memory engine, not a SQL system.

### 2. `.fbrain` is the official format

Always use:

```txt
.fbrain
```

Do not use `.brain`.

### 3. `bbolt` is an internal detail

Do not expose storage internals in the public API.

Users should not need to know about:

- buckets
- KV internals
- page layouts
- `bbolt`

The public abstraction is Brainfile.

### 4. Local-first by default

The project must work locally.

External services are optional.

Tests must pass without internet access and without external model dependencies.

### 5. Embeddings are pluggable

Use the `Embedder` interface.

Supported implementations:

- `DeterministicEmbedder`
- `ExternalEmbedder`
- `E5Embedder`

### 6. Fast Path and Slow Path are distinct

`remember` must stay fast.

Fast Path:

- normalize
- hash
- embed
- simple dedup
- persist

Slow Path:

- extract
- entity/fact enrichment
- relation building
- graph updates
- sync

### 7. FireQuery is contract-bound

FireQuery:

- must not access storage directly
- must call FireMemory through validated contracts
- must reject invalid internal requests

### 8. Internal FireQuery -> FireMemory contracts are in English

If the contract is incomplete or invalid, reject it before calling FireMemory.

### 9. Required validation

Always run:

```sh
go test ./...
```

### 10. Product philosophy

The MVP wins through:

- portability
- semantic deduplication
- context recovery
- traceability
- local-first cognition

It does not win through extreme performance in the MVP.

## Documentation Map

Use this file as the entry point, then follow the relevant references.

### Human-facing documentation

Start here:

- [docs/human/README.md](docs/human/README.md)

### Agent-facing documentation

Start here:

- [docs/agents/README.md](docs/agents/README.md)

### Core implementation references

- Vision: [docs/vision.md](docs/vision.md)
- Architecture: [docs/architecture.md](docs/architecture.md)
- Domain: [docs/domain.md](docs/domain.md)
- Brainfile format: [docs/brainfile-format.md](docs/brainfile-format.md)
- Embeddings: [docs/embedding.md](docs/embedding.md)
- FireMemory MCP: [docs/mcp-integration.md](docs/mcp-integration.md)
- FireQuery architecture: [docs/firequery-architecture.md](docs/firequery-architecture.md)
- FireQuery contract: [docs/firequery-contract.md](docs/firequery-contract.md)
- FireQuery MCP: [docs/firequery-mcp.md](docs/firequery-mcp.md)
- FireQuery runtime: [docs/firequery-runtime.md](docs/firequery-runtime.md)
- Roadmap: [docs/roadmap.md](docs/roadmap.md)
- Tasks: [docs/TASKS.md](docs/TASKS.md)

### Operational references

- Local deploy: [docs/deploy-local.md](docs/deploy-local.md)
- Backup and restore: [docs/backup-restore.md](docs/backup-restore.md)
- Recovery: [docs/recovery.md](docs/recovery.md)
- Compatibility: [docs/compatibility.md](docs/compatibility.md)
- Production hardening: [docs/production-hardening.md](docs/production-hardening.md)
- Release checklists: [docs/release-checklists.md](docs/release-checklists.md)
- Release candidate: [docs/release-candidate.md](docs/release-candidate.md)
- Changelog: [CHANGELOG.md](CHANGELOG.md)

## Precedence

When documents overlap, use this precedence:

1. `AGENTS.md`
2. task-specific spec document
3. implementation and tests
4. `README.md`

If a lower-priority document conflicts with this file, update the documentation rather than inventing behavior.
