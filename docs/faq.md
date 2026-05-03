# FAQ

## General

**What is a `.fbrain` file?**

A single-file memory database for one agent or project. It is a
[bbolt](https://github.com/etcd-io/bbolt) B-tree file with a FireMemory
manifest. You can back it up, copy it between machines, or open multiple
brainfiles in the same session.

**Does FireMemory send data anywhere?**

No. Everything stays local. The only outbound network calls are:
- Downloading ML models on first run (from GitHub Releases / HuggingFace).
- No telemetry, no analytics, no cloud sync.

**Do I need a GPU?**

No. All three models run on CPU with INT8 quantization. A modern laptop handles
inference in under 100 ms per call. GPU acceleration is optional (see
[docs/models.md](models.md)).

**Can I use FireMemory without the ML models?**

The `fmem` CLI (remember, recall, sync, context) works without models using a
deterministic hash-based embedder. Quality is lower — semantic recall and
deduplication are less accurate. The full ONNX pipeline is the intended path.

---

## Installation

**The install script says `~/.local/bin` is not in PATH.**

Add it to your shell profile:
```sh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc   # bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc    # zsh
```
Then reload: `source ~/.bashrc` (or open a new terminal).

**`fquery mcp` hangs on startup.**

It is waiting to download models on first run. Progress is printed to stderr.
If you piped stderr somewhere, open a terminal and run `fquery models pull`
first to pre-download.

**I moved `fquery` to a different directory and it can't find ORT.**

Set `FIREMEMORY_ORT_LIB_PATH` to the absolute path of the ORT shared library:
```sh
export FIREMEMORY_ORT_LIB_PATH=/path/to/libonnxruntime.so
```

---

## Editor integration

**After running `fquery init-mcp cursor`, Cursor still doesn't show FireQuery.**

1. Fully restart Cursor (quit and reopen — not just reload window).
2. Check the config was written: `fquery init-mcp cursor --print`
3. Verify `fquery` is in PATH: `which fquery`
4. Run `fquery doctor` to check model and ORT status.

**Can I use a custom brainfile per project?**

Yes. Set the brainfile path in the MCP server config's `env` block:

```json
{
  "mcpServers": {
    "firequery": {
      "command": "/path/to/fquery",
      "args": ["mcp"],
      "env": {
        "FIREMEMORY_DEFAULT_BRAIN": "/path/to/project.fbrain"
      }
    }
  }
}
```

Or run `fquery init-mcp cursor --print`, edit the output, then write it
manually.

**Which clients does `fquery init-mcp` support?**

`claude-code`, `cursor`, `windsurf`, `zed`. For other clients that support
the standard MCP JSON format, use `--print` to see what would be written and
add it manually.

---

## Data

**How do I back up my brainfile?**

```sh
fmem backup ~/my.fbrain ~/backups/my-$(date +%Y%m%d).fbrain
```

**How do I see what is stored?**

```sh
fmem stats ~/my.fbrain          # memory counts per namespace
fmem snapshot ~/my.fbrain       # full JSON dump
fmem inspect ~/my.fbrain        # manifest info
```

**How do I delete everything and start over?**

Delete the `.fbrain` file and run `fmem init` again. There is no partial delete
yet — use `fmem compact` to reclaim space from soft-deleted records.

**What is `fmem sync` for?**

`remember` is fast: it hashes, embeds, deduplicates, and persists in milliseconds.
`sync` is the slow path: it extracts entities, builds relations, and updates the
knowledge graph. Run it periodically or after batch `remember` operations.

---

## Development

**Tests fail with model-related errors.**

Tests are designed to run offline without models. They use a deterministic
embedder. If you see model errors in tests, you may have set
`FIREQUERY_REQUIRE_REAL_MODELS=1` — unset it for offline test runs.

**How do I run with real models?**

```sh
fquery models pull                  # download once
go test -tags onnx ./...            # run with ONNX backend
```

**Where does the build tag `onnx` come in?**

ONNX inference code is behind `//go:build onnx`. Without the tag, the stub
returns `ErrNotAvailable` and the engine falls back to the deterministic
embedder. Release binaries always include `-tags onnx`.
