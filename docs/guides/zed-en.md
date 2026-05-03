# Zed — Setup Guide

> [Versão em português](zed-pt-BR.md)

This guide shows how to connect **Zed** to FireMemory through the FireQuery MCP server.

## Architecture

```
Zed  →  fquery mcp (stdio)  →  .fbrain file
```

---

## Quick setup (installed binary)

```sh
fquery init-mcp zed
```

That command writes the MCP server entry into Zed's `~/.config/zed/settings.json` and prints the file it modified. Restart Zed and you are done.

### Verify

```sh
fquery init-mcp zed --print   # show what was written
fquery doctor                 # check model and ORT status
```

---

## Manual setup

Add to `~/.config/zed/settings.json` under `"context_servers"`:

```json
{
  "context_servers": {
    "firequery": {
      "command": {
        "path": "fquery",
        "args": ["mcp"]
      }
    }
  }
}
```

### Use a project-specific brainfile

```json
{
  "context_servers": {
    "firequery": {
      "command": {
        "path": "fquery",
        "args": ["mcp"],
        "env": {
          "FIREMEMORY_DEFAULT_BRAIN": "/path/to/project.fbrain"
        }
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

**Zed doesn't show FireQuery tools**
1. Restart Zed.
2. Run `fquery init-mcp zed --print` to verify the config.
3. Run `fquery doctor` — all checks should be green.

**First startup is slow**
- Normal — models (~325 MB) are downloaded once on first run.
