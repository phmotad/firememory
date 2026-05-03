# Agent Documentation

This track is for AI coding agents and automation tooling working inside the repository.

## Primary Entry Point

Read first:

- [AGENTS.md](../../AGENTS.md)

`AGENTS.md` is the canonical scope, rule, and precedence file.

## Delivery Order

Use the mandatory sequence defined in [AGENTS.md](../../AGENTS.md).

## Required Specs

Before implementing or changing behavior, consult the relevant spec:

- Vision: [docs/vision.md](../vision.md)
- Architecture: [docs/reference/architecture.md](../reference/architecture.md)
- Domain: [docs/reference/domain.md](../reference/domain.md)
- Brainfile format: [docs/reference/brainfile-format.md](../reference/brainfile-format.md)
- Embeddings: [docs/reference/embedding.md](../reference/embedding.md)
- FireMemory internal MCP: [docs/reference/mcp-integration.md](../reference/mcp-integration.md)
- FireQuery architecture: [docs/reference/firequery-architecture.md](../reference/firequery-architecture.md)
- FireQuery contract: [docs/reference/firequery-contract.md](../reference/firequery-contract.md)
- FireQuery MCP: [docs/reference/firequery-mcp.md](../reference/firequery-mcp.md)
- FireQuery runtime: [docs/reference/firequery-runtime.md](../reference/firequery-runtime.md)

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

- Compatibility: [docs/operations/compatibility.md](../operations/compatibility.md)
- Production hardening: [docs/operations/production-hardening.md](../operations/production-hardening.md)
- Release checklists: [docs/release/release-checklists.md](../release/release-checklists.md)
- Release candidate: [docs/release/release-candidate.md](../release/release-candidate.md)
