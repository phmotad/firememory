# Cursor — Setup Guide

> [Versão em português](cursor-pt-BR.md)

This guide shows how to connect **Cursor** to FireMemory through the FireQuery MCP server.

## Architecture

```
Cursor  →  fquery mcp (stdio)  →  .fbrain file
```

Cursor talks to **FireQuery**, not directly to FireMemory.

---

## Quick setup (installed binary)

If you installed FireMemory via the install script, Homebrew, or Scoop:

```sh
fquery init-mcp cursor
```

That command writes the MCP server entry into Cursor's config and prints the file it modified. Restart Cursor and you are done.

### Verify

```sh
fquery init-mcp cursor --print   # show what was written
fquery doctor                    # check model and ORT status
```

---

## Manual setup

If you prefer to configure manually, add this to your Cursor MCP config (usually `~/.cursor/mcp.json`):

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

**Windows**
```json
{
  "mcpServers": {
    "firequery": {
      "command": "C:\\Users\\<you>\\AppData\\Local\\firememory\\bin\\fquery.exe",
      "args": ["mcp"]
    }
  }
}
```

### Use a project-specific brainfile

Add an `env` block:

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

## Setup from source

If you are building from source:

```sh
# build
make build

# create a brainfile
./bin/fmem init ./agent.fbrain

# check FireQuery health
./bin/fquery doctor
```

Then add to Cursor's config:

```json
{
  "mcpServers": {
    "firequery": {
      "command": "/absolute/path/to/bin/fquery",
      "args": ["mcp"],
      "cwd": "/absolute/path/to/firememory"
    }
  }
}
```

---

## Available tools

Once connected, Cursor can use:

| Tool | What it does |
|---|---|
| `firequery.remember` | Store a durable memory |
| `firequery.recall` | Semantic search |
| `firequery.get_context` | Ranked context window for a task |
| `firequery.explain` | Debug a memory or retrieval result |
| `firequery.sync` | Run entity/relation enrichment |

---

## Recommended workflow

**Before a large edit or reasoning task** — call `firequery.get_context` with the current task description.

**After an important decision** — call `firequery.remember` to persist it.

**When a recall result looks wrong** — call `firequery.explain` to inspect it.

---

## Troubleshooting

**Cursor doesn't show FireQuery tools**
1. Restart Cursor completely (quit and reopen).
2. Run `fquery init-mcp cursor --print` to verify the config was written.
3. Run `fquery doctor` — all checks should be green.

**`fquery mcp` crashes on startup**
- Run `fquery models list` — models may be missing.
- Run `fquery models pull` to download them.

**First startup is slow**
- Normal — models (~325 MB) are downloaded once on first run.
