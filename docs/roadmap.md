# Roadmap

## Current Status

FireMemory Core is implemented through:

- Brainfile `.fbrain`
- storage abstraction and `BboltStore`
- deterministic embeddings
- linear vector index
- graph engine
- dedup fast path
- remember
- recall
- sync slow path
- context engine
- explain
- CLI `fmem`
- MCP stub

FireQuery is implemented through:

- strict external and internal contracts
- contract validator
- specialist interfaces and heuristic implementations
- runtime detection and CPU fallback
- MCP tool surface
- pipeline integration with FireMemory
- `fquery doctor`
- `fquery devices`

## Next Immediate Step

Cut `v0.1-beta` and continue hardening toward `v1.0`.

## Production Hardening Focus

- storage safety
- backup and restore
- `.fbrain` compatibility and migrations
- structured logging and stable error codes
- concurrency and large-file testing
- MCP contract stabilization
- release and recovery documentation

## Release Path

1. `v0.1-beta`: ready for internal release
2. `v1.0`: production release after additional hardening and compatibility guarantees

## Guardrails

- No SQL interface.
- No external vector database.
- No generative SLM in FireQuery.
- Models suggest, Go validates, FireMemory executes.
