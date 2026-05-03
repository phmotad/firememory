#!/usr/bin/env python3
"""
Export and package FireMemory ONNX INT8 models for release.
"""

from __future__ import annotations

import argparse
import hashlib
import shutil
import sys
import tarfile
import tempfile
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).parent.parent.resolve()

MODELS: list[dict[str, Any]] = [
    {
        "id": "multilingual-e5-small",
        "hf_id": "intfloat/multilingual-e5-small",
        "dir": "multilingual-e5-small",
        "archive": "multilingual-e5-small-onnx-int8.tar.gz",
        "type": "encoder",
    },
    {
        "id": "deberta-v3-small",
        "hf_id": "microsoft/deberta-v3-small",
        "dir": "deberta-v3-small",
        "archive": "deberta-v3-small-onnx-int8.tar.gz",
        "type": "encoder",
    },
    {
        "id": "gliner-small-v2.1",
        "hf_id": "onnx-community/gliner_small-v2.1",
        "dir": "gliner-small-v2.1",
        "archive": "gliner-small-v2.1-onnx-int8.tar.gz",
        "type": "gliner",
    },
]

TOKENIZER_FILES = [
    "tokenizer.json",
    "tokenizer_config.json",
    "special_tokens_map.json",
    "vocab.txt",
    "sentencepiece.bpe.model",
    "spm.model",
    "added_tokens.json",
]

def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()

def _die(msg: str) -> None:
    print(f"\nERROR: {msg}", file=sys.stderr)
    sys.exit(1)

def _report_size(p: Path) -> None:
    print(f"  model.onnx: {p.stat().st_size // 1_000_000} MB")

def export_encoder(hf_id: str, out_dir: Path, quantize: bool) -> None:
    try:
        from optimum.onnxruntime import ORTModelForFeatureExtraction
        from transformers import AutoTokenizer
        from onnxruntime.quantization import quantize_dynamic, QuantType
    except ImportError as e:
        _die(f"Missing dependency: {e}")

    print(f"  Exporting {hf_id} → ONNX fp32...")

    fp32_dir = out_dir / "_fp32"
    fp32_dir.mkdir(parents=True, exist_ok=True)

    model = ORTModelForFeatureExtraction.from_pretrained(hf_id, export=True)
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
        )
    else:
        shutil.copy(fp32_onnx, final_onnx)

    for name in TOKENIZER_FILES:
        src = fp32_dir / name
        if src.exists():
            shutil.copy(src, out_dir / name)

    shutil.rmtree(fp32_dir)

    _report_size(final_onnx)

def export_gliner(hf_id: str, out_dir: Path, quantize: bool) -> None:
    """
    Download prebuilt ONNX GLiNER model from ONNX Community.
    """

    try:
        from huggingface_hub import snapshot_download
    except ImportError:
        _die("huggingface_hub not installed")

    print(f"  Downloading prebuilt ONNX GLiNER model: {hf_id}")

    out_dir.mkdir(parents=True, exist_ok=True)

    snapshot_download(
        repo_id=hf_id,
        local_dir=str(out_dir),
        local_dir_use_symlinks=False,
    )

    onnx_files = list(out_dir.rglob("*.onnx"))

    if not onnx_files:
        _die("No ONNX file found in downloaded GLiNER model")

    model_onnx = onnx_files[0]
    final_onnx = out_dir / "model.onnx"

    if model_onnx.resolve() != final_onnx.resolve():
        shutil.copy(model_onnx, final_onnx)

    print(f"  Using ONNX model: {final_onnx}")

    for p in list(out_dir.rglob("*")):
        if p.is_file():
            keep = (
                p.name.endswith(".onnx")
                or p.name.endswith(".json")
                or p.name in TOKENIZER_FILES
            )

            if not keep:
                p.unlink(missing_ok=True)

    _report_size(final_onnx)

def make_archive(model_dir: Path, dest_dir: Path, dir_name: str, archive_name: str) -> Path:
    dest_dir.mkdir(parents=True, exist_ok=True)

    archive_path = dest_dir / archive_name

    print(f"  Packing {archive_name}...")

    with tarfile.open(archive_path, "w:gz", compresslevel=6) as tf:
        for fp in sorted(model_dir.rglob("*")):
            if fp.is_file():
                relative = fp.relative_to(model_dir)
                tf.add(str(fp), arcname=f"{dir_name}/{relative}")

    print(f"  Archive: {archive_path}")

    return archive_path

def main() -> None:
    parser = argparse.ArgumentParser()

    parser.add_argument("--output-dir", default="dist/models")
    parser.add_argument("--model", action="append", dest="models")
    parser.add_argument("--quantize", default=True, action="store_true")
    parser.add_argument("--no-quantize", dest="quantize", action="store_false")
    parser.add_argument("--kaggle", action="store_true")

    args = parser.parse_args()

    if args.kaggle:
        args.output_dir = "/kaggle/working"
        print("Kaggle mode enabled.")

    output_dir = Path(args.output_dir)

    selected = set(args.models) if args.models else {m["id"] for m in MODELS}

    checksums = {}

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

            archive = make_archive(
                work_dir,
                output_dir,
                spec["dir"],
                spec["archive"],
            )

            digest = sha256_file(archive)
            checksums[spec["archive"]] = digest

            print(f"  SHA256: {digest}")

    sums_path = output_dir / "SHA256SUMS"

    with open(sums_path, "w") as f:
        for name, digest in checksums.items():
            f.write(f"{digest}  {name}\n")

    print(f"\nChecksums → {sums_path}")

if __name__ == "__main__":
    main()
