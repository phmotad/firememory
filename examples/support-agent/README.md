# Support Agent Example

This example models a support agent storing customer incidents and rebuilding context before answering.

## Scenario

- Customer: Joao
- Product: Firebird 2.5
- Problem: fiscal error in NF-e after update 3.2
- Goal: answer the customer with recovered context

## Flow

```sh
go run ./cmd/fmem init ./support-agent.fbrain

go run ./cmd/fmem remember ./support-agent.fbrain "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2"

go run ./cmd/fmem remember ./support-agent.fbrain "Joao relatou novamente problema fiscal em nota eletronica depois da versao 3.2"

go run ./cmd/fmem sync ./support-agent.fbrain

go run ./cmd/fmem context ./support-agent.fbrain "responder Joao sobre erro fiscal apos atualizacao"
```

## What FireMemory Contributes

- Deduplicates repeated operational memories.
- Keeps the customer issue tied to versions and technical terms.
- Builds reusable context before the support agent answers.
- Preserves traceability through persisted traces and sync artifacts.

## Suggested Agent Pattern

1. Store each relevant ticket update with `remember`.
2. Run `sync` before generating a response batch or handoff.
3. Use `context` to build the answer prompt or MCP response payload.
4. Optionally call `inspect` to audit the Brainfile state.
