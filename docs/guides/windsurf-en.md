# Windsurf — Setup Guide

> [Versão em português](windsurf-pt-BR.md)

This guide shows how to connect **Windsurf** to FireMemory through the FireQuery MCP server.

## Architecture

```
Windsurf  →  fquery mcp (stdio)  →  .fbrain file
```

---

## Quick setup (installed binary)

```sh
fquery init-mcp windsurf
```

That command writes the MCP server entry into Windsurf's config and prints the file it modified. Restart Windsurf and you are done.

### Verify

```sh
fquery init-mcp windsurf --print   # show what was written
fquery doctor                      # check model and ORT status
```

---

## Manual setup

Windsurf stores MCP config in `~/.codeium/windsurf/mcp_config.json`:

**macOS / Linux**
```json
{
  "mcpServers": {
    "firequery": {
      "command": "fquery",
      "args": ["mcp"]
    }
  }
}
```

**Windows** (`fquery` requires WSL2 or Docker)
```json
{
  "mcpServers": {
    "firequery": {
      "command": "wsl",
      "args": ["fquery", "mcp"]
    }
  }
}
```

### Use a project-specific brainfile

```json
{
  "mcpServers": {
    "firequery": {
      "command": "fquery",
      "args": ["mcp"],
      "env": {
        "FIREMEMORY_DEFAULT_BRAIN": "/path/to/project.fbrain"
      }
    }
  }
}
```

---

## Available tools

| Tool | What it does |
|---|---|
| `firequery.remember` | Store a durable memory |
| `firequery.recall` | Semantic search |
| `firequery.get_context` | Ranked context window for a task |
| `firequery.explain` | Debug a memory or retrieval result |
| `firequery.sync` | Run entity/relation enrichment |

---

## Troubleshooting

**Windsurf doesn't show FireQuery tools**
1. Restart Windsurf completely.
2. Run `fquery init-mcp windsurf --print` to verify the config.
3. Run `fquery doctor` — all checks should be green.

**First startup is slow**
- Normal — models (~325 MB) are downloaded once on first run.
