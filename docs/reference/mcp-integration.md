# FireMemory Internal MCP Integration

## Overview

FireMemory contains a basic MCP stub after the core engine and CLI are functional.

This layer is intentionally small in the MVP and should be treated as an internal composition layer.

It defines FireMemory tools and routes validated calls to the Go engine.

External agents should not connect to FireMemory directly.

The supported external path is:

Agent -> FireQuery MCP -> FireMemory

## Internal FireMemory Tools

- `firememory.remember`
- `firememory.recall`
- `firememory.get_context`
- `firememory.sync`
- `firememory.explain`

## Design Rules

- MCP stays outside storage internals.
- Tool calls are translated into engine inputs.
- Validation happens before executing the engine operation.
- Responses are structured and deterministic.
- The stub does not implement FireQuery behavior.
- This is not the public MCP entry point for agents.

## Internal Example Tool Call

```json
{
  "tool": "firememory.recall",
  "arguments": {
    "brain_path": "agent.fbrain",
    "query": "erro fiscal NF-e",
    "top_k": 5,
    "include_trace": true
  }
}
```

## Internal Example Tool Response

```json
{
  "tool": "firememory.recall",
  "ok": true,
  "content": {
    "hits": [],
    "trace": [
      "embedded recall query",
      "executed vector search"
    ]
  }
}
```

## MVP Scope

The MVP MCP layer provides:

- tool registry
- input schemas
- call dispatcher
- basic integration tests

For external transport today, prefer FireQuery's standalone MCP server:

```sh
./bin/fquery mcp
```

It does not yet provide:

- transport server
- external authentication
- FireQuery contracts
- model orchestration
