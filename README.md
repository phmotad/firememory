# FireMemory

[![Test](https://github.com/phmotad/firememory/actions/workflows/test.yml/badge.svg)](https://github.com/phmotad/firememory/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/phmotad/firememory)](https://github.com/phmotad/firememory/releases/latest)
[![License](https://img.shields.io/github/license/phmotad/firememory)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/phmotad/firememory)](https://goreportcard.com/report/github.com/phmotad/firememory)

**Local-first semantic memory engine for AI agents.**

FireMemory stores everything in a single `.fbrain` file — no server, no cloud, no configuration.
Agents read and write memory through [MCP](https://modelcontextprotocol.io/) via `fquery mcp`.
ML models (~280 MB) are downloaded automatically on first use.

---

## 60-second quickstart

### 1. Install

**macOS / Linux**
```sh
curl -fsSL https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.sh | bash
```

**Windows** (PowerShell)
```powershell
irm https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.ps1 | iex
```

**Homebrew**
```sh
brew tap phmotad/firememory
brew install firememory
```

**Scoop**
```powershell
scoop bucket add phmotad https://github.com/phmotad/scoop-firememory
scoop install firememory
```

### 2. Wire your editor

```sh
fquery init-mcp claude-code   # Claude Code
fquery init-mcp cursor        # Cursor
fquery init-mcp windsurf      # Windsurf
fquery init-mcp zed           # Zed
```

This writes the MCP server entry into the editor's config file and prints the path it modified.

### 3. Create a brainfile

```sh
fmem init ~/my.fbrain
```

Or skip this — `fmem stats` and any `fquery` tool call will auto-create
`~/.firememory/default.fbrain` if it doesn't exist.

### 4. Restart your editor

The MCP server starts on demand. On the first call, `fquery mcp` downloads the
three ML models (~280 MB, runs once). Subsequent starts are instant.

---

## What it is

FireMemory is **not** a vector database, **not** a RAG layer, and **not** SQL.

It is a *cognitive memory engine*: it understands what is being stored, deduplicates
semantically, builds a knowledge graph, and assembles context windows tailored to a query.

| Concept | FireMemory |
|---|---|
| Storage format | Single `.fbrain` file (bbolt) |
| Embeddings | multilingual-e5-small INT8 (local ONNX) |
| Entity extraction | GLiNER-small-v2.1 INT8 (local ONNX) |
| Intent / classification | DeBERTa-v3-small INT8 (local ONNX) |
| Model size | ~280 MB total, downloaded once |
| Transport | MCP over stdio (`fquery mcp`) |
| Privacy | 100% local — nothing leaves your machine |

---

## Agent connectivity

Agents talk to **FireQuery** (the MCP layer), not directly to FireMemory.

```
Your editor agent
      │  MCP (stdio)
      ▼
  fquery mcp          ← FireQuery: validates, classifies, enriches
      │
      ▼
  .fbrain file        ← FireMemory: stores, recalls, syncs
```

### Supported MCP tools

| Tool | Description |
|---|---|
| `remember` | Store a memory (deduplication is automatic) |
| `recall` | Semantic search over stored memories |
| `get_context` | Retrieve a ranked context window for a query |
| `sync` | Run slow-path enrichment (entities, relations, graph) |
| `explain` | Explain a stored memory |

---

## CLI reference

### fmem

```
fmem init <file.fbrain>                 create a new brainfile
fmem remember <file.fbrain> <text>      store a memory
fmem recall <file.fbrain> <query>       semantic search
fmem sync <file.fbrain>                 entity/relation enrichment
fmem context <file.fbrain> <query>      build a context window
fmem inspect <file.fbrain>              show manifest
fmem snapshot <file.fbrain>             full data dump (JSON)
fmem backup <file.fbrain> <dest>        copy to backup path
fmem restore <backup> <file.fbrain>     restore from backup
fmem compact <file.fbrain>              reclaim space (bbolt vacuum)
fmem stats [<file.fbrain>]              memory counts
fmem default                            print/create default brainfile path
fmem version                            print version
```

### fquery

```
fquery mcp                              start MCP server (stdio)
fquery init-mcp <client>               configure editor MCP entry
  clients: claude-code, cursor, windsurf, zed
  --print                               dry-run: show config that would be written
  --config <path>                       override config file path
fquery models list                      show downloaded model status
fquery models pull                      download missing models
fquery models pull --force              re-download all models
fquery models gc                        remove cached models
fquery devices                          list compute devices (CPU/GPU)
fquery doctor                           run diagnostics
fquery version                          print version
```

---

## Models

FireQuery uses three local ONNX INT8 models, downloaded automatically:

| Model | Use | Size |
|---|---|---|
| `multilingual-e5-small` | Embeddings, semantic recall | ~120 MB |
| `deberta-v3-small` | Intent & trigger classification | ~72 MB |
| `gliner-small-v2.1` | Named entity extraction | ~90 MB |

Models are stored in:
- **macOS** — `~/Library/Caches/firememory/models`
- **Linux** — `~/.cache/firememory/models`
- **Windows** — `%LOCALAPPDATA%\firememory\models`

Override with `FIREMEMORY_MODELS_DIR`.

To remove: `fquery models gc`

---

## Docker

```sh
docker run --rm -i \
  -v "$HOME/.firememory/models:/models" \
  ghcr.io/phmotad/firequery mcp
```

Models are cached in the mounted volume and downloaded on first run.

---

## Build from source

Requires Go 1.24 and a C compiler (for CGO).

```sh
git clone https://github.com/phmotad/firememory
cd firememory
make build          # produces bin/fmem and bin/fquery (with -tags onnx)
make test           # runs all tests (offline-safe, no models needed)
```

Release binaries are built with `goreleaser` and the ONNX Runtime shared library
is bundled in each archive (no separate install needed).

---

## Architecture

```
cmd/fmem       — FireMemory CLI
cmd/fquery     — FireQuery CLI + MCP server

internal/
  engine/        — remember / recall / sync / context / explain
  storage/       — bbolt store behind the Store interface
  brainfile/     — .fbrain format, validation, migration
  dedup/         — semantic deduplication (hash + embedding)
  embedder/      — Embedder interface (E5, deterministic, external)
  graph/         — knowledge graph (entities + relations)
  firequery/     — cognitive interface layer (pipeline, MCP, contracts)
  firequery/onnx — ONNX inference backend (build tag: onnx)
  modelcache/    — auto-download, verify, extract ML models
  initcfg/       — write MCP entries into editor config files
  defaultbrain/  — default brainfile path + auto-init
  version/       — version string injected at build time
```

Fast path (`remember`): hash → embed → dedup → persist  
Slow path (`sync`): extract entities → build relations → update graph

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). All tests must pass (`go test ./...`) before submitting a PR.

The ONNX backend is behind `//go:build onnx` — tests run offline without models by design.

---

## License

MIT — see [LICENSE](LICENSE).
