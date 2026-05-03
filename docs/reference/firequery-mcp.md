# FireQuery MCP

## Purpose

FireQuery exposes an MCP surface for external agents.

This is the supported MCP entry point for agents.

External flow:

Agent -> FireQuery MCP -> FireMemory

External communication may use the user language.

Internal communication with FireMemory must be in English.

The repository now ships a standalone `stdio` MCP server through:

```sh
./bin/fquery mcp
```

## External tools

The active tool surface is:

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## Tool behavior

### `firequery.ask`

General cognitive entry point.

FireQuery infers a safe operation if the request does not specify one.

Typical resolution order:

- `remember` when `input.content` is present
- `explain` when `input.target_operation` is present
- `get_context` when `input.task` is present
- `recall` otherwise

### `firequery.plan`

Produces a structured plan for memory-aware execution without writing to FireMemory.

The MCP wrapper forces:

- `operation = "get_context"`
- `input.planning_mode = true`
- `input.allow_write = false`

### `firequery.remember`

Forces `operation = "remember"`.

### `firequery.recall`

Forces `operation = "recall"`.

### `firequery.get_context`

Forces `operation = "get_context"`.

### `firequery.explain`

Forces `operation = "explain"`.

## Schemas

Schemas are defined in:

- `internal/firequery/mcp/schemas.go`

Each tool schema includes:

- required top-level fields
- tool-specific input fields
- standard structured output fields

## External contract requirements

Every MCP call should include:

- `version`
- `request_id`
- `actor`
- `operation`
- `brain`
- operation-specific input

Language may be user-facing.

The external contract may be more permissive than the internal one, but it must still be validated.

## Registration

Default MCP tool registration is available through:

- `(*mcp.Server).RegisterDefaultTools(...)`

This registers all six official FireQuery tools and applies the expected operation wrapper for each one.

## Transport

The transport currently implemented is:

- `stdio` JSON-RPC for MCP clients

The server entry point is:

- `cmd/fquery`
- command: `fquery mcp`

This is the integration path for tools such as Claude Code, Codex, Cursor, and other MCP-capable clients.

## Examples

Examples are available in:

- `examples/firequery-mcp/README.md`
- `examples/mcp-clients/README.md`

## Response shape

Responses should be structured and machine-friendly:

```json
{
  "ok": true,
  "request_id": "req_123",
  "operation": "recall",
  "data": {},
  "trace": {}
}
```

On validation failure:

```json
{
  "ok": false,
  "request_id": "req_123",
  "rejected": true,
  "error": {
    "code": "EXTERNAL_CONTRACT_VALIDATION_FAILED",
    "message": "Missing required field: brain"
  }
}
```
