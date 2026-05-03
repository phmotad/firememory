# Claude Code — Setup Guide

> [Versão em português](claude-code-pt-BR.md)

This guide shows how to connect **Claude Code** to FireMemory through the FireQuery MCP server.

## Architecture

```
Claude Code  →  fquery mcp (stdio)  →  .fbrain file
```

Claude Code talks to **FireQuery**, not directly to FireMemory.

---

## Quick setup (installed binary)

If you installed FireMemory via the install script, Homebrew, or Scoop:

```sh
fquery init-mcp claude-code
```

That command writes the MCP server entry into Claude Code's config (`~/.claude/settings.json`) and prints the file it modified. Restart Claude Code and you are done.

### Verify

```sh
fquery init-mcp claude-code --print   # show what was written
fquery doctor                         # check model and ORT status
```

---

## Manual setup

Add to `~/.claude/settings.json`:

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

**Windows** (note: `fquery` requires WSL2 or Docker on Windows)
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

Then add to `~/.claude/settings.json`:

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

## AGENTS.md integration

Point Claude Code to the project's `AGENTS.md` for canonical project instructions. Claude Code reads this file automatically when present in the repository root.

---

## Available tools

Once connected, Claude Code can use:

| Tool | What it does |
|---|---|
| `firequery.remember` | Store a durable memory |
| `firequery.recall` | Semantic search |
| `firequery.get_context` | Ranked context window for a task |
| `firequery.explain` | Debug a memory or retrieval result |
| `firequery.sync` | Run entity/relation enrichment |

---

## Recommended workflow

**Before a large reasoning task or code change** — call `firequery.get_context` with the current task.

**After a durable decision or user fact** — call `firequery.remember`.

**When a recall result looks wrong** — call `firequery.explain`.

---

## Troubleshooting

**Claude Code doesn't show FireQuery tools**
1. Restart Claude Code.
2. Run `fquery init-mcp claude-code --print` to verify the config.
3. Run `fquery doctor` — all checks should be green.

**`fquery mcp` crashes on startup**
- Run `fquery models pull` to download missing models.

**First startup is slow**
- Normal — models (~325 MB) are downloaded once on first run.
