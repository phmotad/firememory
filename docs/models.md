# Models

FireQuery uses three local ONNX INT8 models. They are downloaded automatically
on the first call to `fquery mcp` and never leave your machine.

## The three models

| ID | File | Purpose | Size |
|---|---|---|---|
| `multilingual-e5-small` | `multilingual-e5-small-onnx-int8.tar.gz` | Embedding (similarity, recall) | ~120 MB |
| `deberta-v3-small` | `deberta-v3-small-onnx-int8.tar.gz` | Intent & trigger classification | ~72 MB |
| `gliner-small-v2.1` | `gliner-small-v2.1-onnx-int8.tar.gz` | Named entity extraction (NER) | ~90 MB |

Total compressed: ~282 MB. Extracted on disk: slightly more.

## Cache location

| Platform | Default path |
|---|---|
| macOS | `~/Library/Caches/firememory/models` |
| Linux | `$XDG_CACHE_HOME/firememory/models` or `~/.cache/firememory/models` |
| Windows | `%LOCALAPPDATA%\firememory\models` |

Override with the `FIREMEMORY_MODELS_DIR` environment variable.

## Managing models

```sh
# Show download status and cache location
fquery models list

# Download any missing models
fquery models pull

# Force re-download all models (replaces existing files)
fquery models pull --force

# Remove all cached model files
fquery models gc
```

## Integrity verification

Each archive is verified with SHA256 after download. The expected hash is
embedded in the `fquery` binary at build time (`internal/modelcache/manifest.json`).

A stored `.sha256` file next to each model directory is checked on subsequent
starts. If the file is missing or the hash does not match, the model is
re-downloaded automatically.

## ONNX Runtime

The ONNX Runtime shared library (`libonnxruntime.so` / `.dylib` / `.dll`) is
bundled in every release archive and placed next to the binary. `fquery` loads
it at startup via `FIREMEMORY_ORT_LIB_PATH` or by looking next to its own
executable.

If you built from source, set `FIREMEMORY_ORT_LIB_PATH` to point to your ORT
installation, or download version 1.20.0 from
<https://github.com/microsoft/onnxruntime/releases>.

## Hardware acceleration

By default, inference runs on CPU. GPU acceleration can be enabled via
environment variables:

| Variable | Backend |
|---|---|
| `FIREQUERY_ENABLE_CUDA=1` | NVIDIA CUDA |
| `FIREQUERY_ENABLE_DIRECTML=1` | DirectML (Windows) |
| `FIREQUERY_ENABLE_COREML=1` | CoreML (macOS) |
| `FIREQUERY_ENABLE_OPENVINO=1` | Intel OpenVINO |

These require matching ORT execution providers and hardware. Use
`fquery devices` to see what is available on your machine.
