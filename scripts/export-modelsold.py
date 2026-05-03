#!/usr/bin/env python3
"""
Export and package FireMemory ONNX INT8 models for release.

Steps performed:
  1. Download each model from HuggingFace
  2. Export to ONNX (fp32)
  3. Apply INT8 dynamic quantization
  4. Package into .tar.gz (structure expected by modelcache/extract.go)
  5. Compute SHA256 checksums
  6. Optionally patch internal/modelcache/manifest.json
  7. Optionally upload to GitHub Releases via `gh`

Usage:
    python scripts/export-models.py [options]

    # Full run — export, quantize, update manifest, upload
    python scripts/export-models.py --update-manifest --upload

    # Export a single model (for debugging)
    python scripts/export-models.py --model multilingual-e5-small

    # Skip quantization (faster, larger files)
    python scripts/export-models.py --no-quantize

Requirements (install once):
    pip install torch transformers "optimum[onnxruntime]" onnxruntime gliner huggingface_hub
"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path
from typing import Any

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

REPO_ROOT     = Path(__file__).parent.parent.resolve()
MANIFEST_PATH = REPO_ROOT / "internal" / "modelcache" / "manifest.json"

MODELS: list[dict[str, Any]] = [
    {
        "id":       "multilingual-e5-small",
        "hf_id":    "intfloat/multilingual-e5-small",
        "dir":      "multilingual-e5-small",
        "archive":  "multilingual-e5-small-onnx-int8.tar.gz",
        "type":     "encoder",
    },
    {
        "id":       "deberta-v3-small",
        "hf_id":    "microsoft/deberta-v3-small",
        "dir":      "deberta-v3-small",
        "archive":  "deberta-v3-small-onnx-int8.tar.gz",
        "type":     "encoder",
    },
    {
        "id":       "gliner-small-v2.1",
        "hf_id":    "urchade/gliner_small-v2.1",
        "dir":      "gliner-small-v2.1",
        "archive":  "gliner-small-v2.1-onnx-int8.tar.gz",
        "type":     "gliner",
    },
]

# Tokenizer files we want to keep alongside model.onnx in the archive.
TOKENIZER_FILES = [
    "tokenizer.json",
    "tokenizer_config.json",
    "special_tokens_map.json",
    "vocab.txt",
    "sentencepiece.bpe.model",
    "spm.model",
    "added_tokens.json",
]

# ---------------------------------------------------------------------------
# SHA256 helper
# ---------------------------------------------------------------------------

def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()

# ---------------------------------------------------------------------------
# Encoder export  (E5, DeBERTa — standard HF transformer)
# ---------------------------------------------------------------------------

def export_encoder(hf_id: str, out_dir: Path, quantize: bool) -> None:
    try:
        from optimum.onnxruntime import ORTModelForFeatureExtraction
        from transformers import AutoTokenizer
        from onnxruntime.quantization import quantize_dynamic, QuantType
    except ImportError as e:
        _die(f"Missing dependency: {e}\nRun: pip install 'optimum[onnxruntime]' transformers onnxruntime")

    print(f"  Exporting {hf_id} → ONNX fp32...")
    fp32_dir = out_dir / "_fp32"
    fp32_dir.mkdir(parents=True, exist_ok=True)

    model     = ORTModelForFeatureExtraction.from_pretrained(hf_id, export=True)
    tokenizer = AutoTokenizer.from_pretrained(hf_id)
    model.save_pretrained(str(fp32_dir))
    tokenizer.save_pretrained(str(fp32_dir))

    fp32_onnx = fp32_dir / "model.onnx"
    final_onnx = out_dir / "model.onnx"

    if quantize:
        print("  Quantizing to INT8 (dynamic)...")
        quantize_dynamic(
            str(fp32_onnx),
            str(final_onnx),
            weight_type=QuantType.QInt8,
            optimize_model=True,
        )
    else:
        shutil.copy(fp32_onnx, final_onnx)

    # Copy tokenizer files needed at inference time.
    for name in TOKENIZER_FILES:
        src = fp32_dir / name
        if src.exists():
            shutil.copy(src, out_dir / name)

    shutil.rmtree(fp32_dir)
    _report_size(final_onnx)

# ---------------------------------------------------------------------------
# GLiNER export
# ---------------------------------------------------------------------------

def export_gliner(hf_id: str, out_dir: Path, quantize: bool) -> None:
    try:
        from gliner import GLiNER
    except ImportError:
        _die("gliner package not installed.\nRun: pip install gliner")

    try:
        from transformers import AutoTokenizer
        from onnxruntime.quantization import quantize_dynamic, QuantType
    except ImportError as e:
        _die(f"Missing dependency: {e}")

    print(f"  Loading GLiNER {hf_id}...")
    model     = GLiNER.from_pretrained(hf_id)
    tokenizer = AutoTokenizer.from_pretrained(hf_id)
    out_dir.mkdir(parents=True, exist_ok=True)

    fp32_onnx  = out_dir / "model_fp32.onnx"
    final_onnx = out_dir / "model.onnx"

    # Try GLiNER's built-in export (available in gliner >= 0.2.x).
    exported = _try_gliner_builtin_export(model, out_dir, fp32_onnx)

    if not exported:
        print("  Built-in ONNX export not available; using torch.onnx.export...")
        _export_gliner_via_torch(model, fp32_onnx)

    if quantize:
        print("  Quantizing GLiNER to INT8 (dynamic)...")
        quantize_dynamic(
            str(fp32_onnx),
            str(final_onnx),
            weight_type=QuantType.QInt8,
            optimize_model=True,
        )
        fp32_onnx.unlink(missing_ok=True)
    else:
        fp32_onnx.rename(final_onnx)

    tokenizer.save_pretrained(str(out_dir))

    # Remove any extra directories created by built-in export.
    for p in out_dir.iterdir():
        if p.is_dir():
            shutil.rmtree(p)

    _report_size(final_onnx)


def _try_gliner_builtin_export(model: Any, out_dir: Path, fp32_onnx: Path) -> bool:
    """
    Attempt GLiNER's own ONNX export method.
    Returns True if a model.onnx (or equivalent) was produced.
    """
    tmp = out_dir / "_gliner_export_tmp"
    tmp.mkdir(exist_ok=True)
    try:
        if hasattr(model, "save_pretrained_onnx"):
            model.save_pretrained_onnx(str(tmp))
        elif hasattr(model, "export_to_onnx"):
            model.export_to_onnx(str(tmp / "model.onnx"))
        else:
            shutil.rmtree(tmp, ignore_errors=True)
            return False

        # Find the produced .onnx file.
        candidates = list(tmp.rglob("*.onnx"))
        if not candidates:
            shutil.rmtree(tmp, ignore_errors=True)
            return False

        shutil.move(str(candidates[0]), str(fp32_onnx))
        shutil.rmtree(tmp, ignore_errors=True)
        return True
    except Exception as exc:
        print(f"  Built-in export raised {exc!r}, falling back to torch.onnx.export...")
        shutil.rmtree(tmp, ignore_errors=True)
        return False


def _export_gliner_via_torch(model: Any, out_path: Path) -> None:
    """
    Manual torch.onnx.export for GLiNER using the input signature expected by
    internal/firequery/onnx/gliner.go:
      inputs  — input_ids, attention_mask, words_mask, text_lengths
      outputs — logits [batch, maxWords, maxWords, numTypes]
    """
    try:
        import torch
    except ImportError:
        _die("torch not installed.\nRun: pip install torch")

    inner = model.model
    inner.eval()
    device  = next(inner.parameters()).device
    max_len = int(getattr(model, "max_len", getattr(model, "max_length", 384)))

    # Minimal dummy batch: seq_len=max_len, 1 word, 1 entity type.
    dummy_ids        = torch.zeros(1, max_len, dtype=torch.long, device=device)
    dummy_attn       = torch.ones(1,  max_len, dtype=torch.long, device=device)
    dummy_words_mask = torch.zeros(1, max_len, dtype=torch.long, device=device)
    dummy_words_mask[0, 1] = 1          # first word at token position 1
    dummy_text_len   = torch.tensor([1], dtype=torch.long, device=device)

    torch.onnx.export(
        inner,
        (dummy_ids, dummy_attn, dummy_words_mask, dummy_text_len),
        str(out_path),
        input_names  = ["input_ids", "attention_mask", "words_mask", "text_lengths"],
        output_names = ["logits"],
        dynamic_axes = {
            "input_ids":      {0: "batch", 1: "seq_len"},
            "attention_mask": {0: "batch", 1: "seq_len"},
            "words_mask":     {0: "batch", 1: "seq_len"},
            "text_lengths":   {0: "batch"},
            "logits":         {0: "batch", 1: "num_words", 2: "num_words"},
        },
        opset_version = 14,
        do_constant_folding = True,
    )

# ---------------------------------------------------------------------------
# Archive creation
# ---------------------------------------------------------------------------

def make_archive(model_dir: Path, dest_dir: Path, dir_name: str, archive_name: str) -> Path:
    """
    Pack model_dir contents into <dest_dir>/<archive_name>.

    Archive layout: <dir_name>/<file> for every file in model_dir.
    modelcache/extract.go strips directory prefixes so all files land flat in
    cacheDir/<dir_name>/ at extraction time.
    """
    dest_dir.mkdir(parents=True, exist_ok=True)
    archive_path = dest_dir / archive_name

    print(f"  Packing {archive_name}...")
    with tarfile.open(archive_path, "w:gz", compresslevel=6) as tf:
        for fp in sorted(model_dir.iterdir()):
            if fp.is_file():
                tf.add(str(fp), arcname=f"{dir_name}/{fp.name}")

    print(f"  Archive: {archive_path} ({archive_path.stat().st_size // 1_000_000} MB)")
    return archive_path

# ---------------------------------------------------------------------------
# Manifest patcher
# ---------------------------------------------------------------------------

def update_manifest(checksums: dict[str, tuple[str, int]]) -> None:
    with open(MANIFEST_PATH) as f:
        manifest = json.load(f)

    changed = 0
    for entry in manifest["models"]:
        if entry["archive"] in checksums:
            sha256, size = checksums[entry["archive"]]
            entry["sha256"]           = sha256
            entry["compressed_bytes"] = size
            changed += 1
            print(f"  {entry['id']}: sha256={sha256[:16]}...  size={size // 1_000_000} MB")

    with open(MANIFEST_PATH, "w") as f:
        json.dump(manifest, f, indent=2)
        f.write("\n")

    print(f"  Wrote {MANIFEST_PATH}  ({changed} entries updated)")

# ---------------------------------------------------------------------------
# GitHub upload
# ---------------------------------------------------------------------------

def upload_to_github(archives: list[Path], tag: str) -> None:
    if not shutil.which("gh"):
        _die("`gh` CLI not found.\nInstall from https://cli.github.com/ and run `gh auth login`.")

    # Create the release if it does not exist.
    result = subprocess.run(["gh", "release", "view", tag], capture_output=True)
    if result.returncode != 0:
        print(f"  Creating release '{tag}'...")
        subprocess.run(
            ["gh", "release", "create", tag,
             "--title", f"ML models ({tag})",
             "--notes", "ONNX INT8 model archives. Downloaded automatically by `fquery mcp`.",
             "--prerelease"],
            check=True,
        )

    for archive in archives:
        print(f"  Uploading {archive.name}...")
        subprocess.run(
            ["gh", "release", "upload", tag, str(archive), "--clobber"],
            check=True,
        )
    print(f"  Done — archives available at GitHub Release '{tag}'.")

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _report_size(p: Path) -> None:
    print(f"  model.onnx: {p.stat().st_size // 1_000_000} MB")


def _die(msg: str) -> None:
    print(f"\nERROR: {msg}", file=sys.stderr)
    sys.exit(1)

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--output-dir", default="dist/models",
        help="Directory for output archives (default: dist/models)",
    )
    parser.add_argument(
        "--model", action="append", dest="models", metavar="ID",
        help="Export only this model ID; repeat for multiple (default: all)",
    )
    parser.add_argument(
        "--quantize", default=True, action="store_true",
        help="Apply INT8 dynamic quantization (default)",
    )
    parser.add_argument(
        "--no-quantize", dest="quantize", action="store_false",
        help="Skip quantization (faster export, larger files)",
    )
    parser.add_argument(
        "--update-manifest", action="store_true",
        help=f"Patch {MANIFEST_PATH.relative_to(REPO_ROOT)} with computed SHA256 values",
    )
    parser.add_argument(
        "--upload", action="store_true",
        help="Upload archives to GitHub Releases via `gh`",
    )
    parser.add_argument(
        "--tag", default="models-v1",
        help="GitHub Release tag to upload to (default: models-v1)",
    )
    parser.add_argument(
        "--kaggle", action="store_true",
        help="Kaggle mode: output to /kaggle/working/, skip upload, print SHA256 for manual copy",
    )
    args = parser.parse_args()

    # Kaggle shortcut: override output dir and disable upload.
    if args.kaggle:
        args.output_dir   = "/kaggle/working"
        args.upload       = False
        args.update_manifest = False
        print("Kaggle mode: archives → /kaggle/working/  (download from Kaggle output panel)")

    output_dir = REPO_ROOT / args.output_dir
    selected   = set(args.models) if args.models else {m["id"] for m in MODELS}

    archives:   list[Path]                     = []
    checksums:  dict[str, tuple[str, int]]     = {}  # archive_name → (sha256, bytes)

    with tempfile.TemporaryDirectory(prefix="firememory-export-") as tmp:
        tmp_path = Path(tmp)

        for spec in MODELS:
            if spec["id"] not in selected:
                continue

            print(f"\n{'─'*60}")
            print(f"  Model: {spec['id']}")
            print(f"{'─'*60}")

            work_dir = tmp_path / spec["dir"]
            work_dir.mkdir(parents=True, exist_ok=True)

            if spec["type"] == "gliner":
                export_gliner(spec["hf_id"], work_dir, args.quantize)
            else:
                export_encoder(spec["hf_id"], work_dir, args.quantize)

            archive = make_archive(work_dir, output_dir, spec["dir"], spec["archive"])
            archives.append(archive)

            digest = sha256_file(archive)
            size   = archive.stat().st_size
            checksums[spec["archive"]] = (digest, size)
            print(f"  SHA256: {digest}")

    # Write SHA256SUMS file alongside the archives.
    sums_path = output_dir / "SHA256SUMS"
    with open(sums_path, "w") as f:
        for name, (digest, _) in checksums.items():
            f.write(f"{digest}  {name}\n")
    print(f"\nChecksums → {sums_path}")

    # Patch manifest.json if requested.
    if args.update_manifest:
        print(f"\nUpdating manifest.json...")
        update_manifest(checksums)

    # Upload to GitHub Releases if requested.
    if args.upload:
        print(f"\nUploading to GitHub Release '{args.tag}'...")
        upload_to_github(archives, args.tag)

    # Final summary.
    print(f"\n{'═'*60}")
    print("  SUMMARY")
    print(f"{'═'*60}")
    for name, (digest, size) in checksums.items():
        print(f"  {name}")
        print(f"    sha256 = {digest}")
        print(f"    size   = {size // 1_000_000} MB")

    hints = []
    if not args.update_manifest:
        hints.append("--update-manifest  (patch manifest.json with real SHA256 values)")
    if not args.upload:
        hints.append("--upload           (upload archives to GitHub Releases)")
    if hints:
        print(f"\nTip — re-run with:")
        for h in hints:
            print(f"  {h}")

    if args.update_manifest and not args.upload:
        print(
            "\nNext steps:\n"
            "  1. Commit the updated manifest.json\n"
            "  2. Push and create a git tag (e.g. git tag v0.1.0 && git push --tags)\n"
            "  3. GoReleaser will build and publish the binaries automatically"
        )


if __name__ == "__main__":
    main()
