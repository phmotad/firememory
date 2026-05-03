# Agent Documentation

This track is for AI coding agents and automation tooling working inside the repository.

## Primary Entry Point

Read first:

- [AGENTS.md](AGENTS.md)

`AGENTS.md` is the canonical scope, rule, and precedence file.

## Delivery Order

Use the mandatory sequence defined in:

- [AGENTS.md](AGENTS.md)
- [docs/TASKS.md](docs/TASKS.md)

## Required Specs

Before implementing or changing behavior, consult the relevant spec:

- Vision: [docs/vision.md](docs/vision.md)
- Architecture: [docs/architecture.md](docs/architecture.md)
- Domain: [docs/domain.md](docs/domain.md)
- Brainfile format: [docs/brainfile-format.md](docs/brainfile-format.md)
- Embeddings: [docs/embedding.md](docs/embedding.md)
- FireMemory internal MCP: [docs/mcp-integration.md](docs/mcp-integration.md)
- FireQuery architecture: [docs/firequery-architecture.md](docs/firequery-architecture.md)
- FireQuery contract: [docs/firequery-contract.md](docs/firequery-contract.md)
- FireQuery MCP: [docs/firequery-mcp.md](docs/firequery-mcp.md)
- FireQuery runtime: [docs/firequery-runtime.md](docs/firequery-runtime.md)

## Validation Rules

Minimum validation:

```sh
go test ./...
```

When changing runtime behavior, also inspect:

- examples
- CLI coverage
- contract validation paths
- recovery and compatibility docs

## Guardrails

- Do not introduce SQL or SQL-like APIs.
- Do not expose `bbolt` in the public API.
- Do not use `.brain`; use `.fbrain`.
- Do not let FireQuery bypass FireMemory.
- Do not let models write directly.
- Preserve local-first operation.

## Operational References

- Compatibility: [docs/compatibility.md](docs/compatibility.md)
- Production hardening: [docs/production-hardening.md](docs/production-hardening.md)
- Release checklists: [docs/release-checklists.md](docs/release-checklists.md)
- Release candidate: [docs/release-candidate.md](docs/release-candidate.md)
