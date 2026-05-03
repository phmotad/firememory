# MCP Client Registration Examples

This folder shows the supported agent integration pattern:

Agent -> FireQuery MCP -> FireMemory

Do not register FireMemory directly as the public MCP endpoint.

## Required command

Build the binary:

```sh
go build -o ./bin/fquery ./cmd/fquery
```

Start the MCP server:

```sh
./bin/fquery mcp
```

On Windows:

```powershell
.\bin\fquery.exe mcp
```

## Generic stdio registration

Most MCP-capable clients need the same command definition:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "C:\\Projects\\FireMemory\\bin\\fquery.exe",
    "args": ["mcp"],
    "cwd": "C:\\Projects\\FireMemory"
  }
}
```

On Unix-like systems:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "/path/to/FireMemory/bin/fquery",
    "args": ["mcp"],
    "cwd": "/path/to/FireMemory"
  }
}
```

## Claude Code

If your Claude Code setup supports MCP server registration by command, use:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "C:\\Projects\\FireMemory\\bin\\fquery.exe",
    "args": ["mcp"],
    "cwd": "C:\\Projects\\FireMemory"
  }
}
```

Recommended workflow:

1. Keep one `.fbrain` per project or agent.
2. Use `firequery.get_context` before long reasoning or coding turns.
3. Use `firequery.remember` only when you want durable memory.

## Codex

If your Codex environment supports MCP command registration, use the same stdio command:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "C:\\Projects\\FireMemory\\bin\\fquery.exe",
    "args": ["mcp"],
    "cwd": "C:\\Projects\\FireMemory"
  }
}
```

Practical rule:

- Codex should talk to FireQuery
- FireQuery should talk to FireMemory

## Cursor

If Cursor MCP is enabled in your installation, register this command:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "C:\\Projects\\FireMemory\\bin\\fquery.exe",
    "args": ["mcp"],
    "cwd": "C:\\Projects\\FireMemory"
  }
}
```

Recommended usage:

- `firequery.ask` for general retrieval-aware requests
- `firequery.get_context` before large edits
- `firequery.explain` for debugging memory decisions

## Antigravity

If Antigravity can attach to MCP over `stdio`, use the same command registration:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "C:\\Projects\\FireMemory\\bin\\fquery.exe",
    "args": ["mcp"],
    "cwd": "C:\\Projects\\FireMemory"
  }
}
```

If it cannot, wrap `fquery mcp` in the integration layer Antigravity expects.

## Useful tool names

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## Example request

```json
{
  "version": "0.1",
  "request_id": "req_context_001",
  "language": "pt-BR",
  "actor": {
    "type": "agent",
    "id": "support-agent"
  },
  "brain": "./agent.fbrain",
  "input": {
    "task": "responder Joao sobre erro fiscal apos atualizacao",
    "budget_tokens": 1500
  }
}
```

## Notes

- Client-specific config file names and UI screens may vary by product version.
- The stable part on the FireMemory side is the command: `fquery mcp`.
- The supported external architecture is always FireQuery first.
