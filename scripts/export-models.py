#!/usr/bin/env python3
"""
Export and package FireMemory ONNX INT8 models for release.
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
    try:
        from gliner import GLiNER
        from onnxruntime.quantization import quantize_dynamic, QuantType
    except ImportError as e:
        _die(f"Missing dependency: {e}")

    print(f"  Loading GLiNER {hf_id}...")

    model = GLiNER.from_pretrained(hf_id)
    tokenizer = model.data_processor.transformer_tokenizer

    out_dir.mkdir(parents=True, exist_ok=True)

    fp32_onnx  = out_dir / "model_fp32.onnx"
    final_onnx = out_dir / "model.onnx"

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
        )
        fp32_onnx.unlink(missing_ok=True)
    else:
        fp32_onnx.rename(final_onnx)

    tokenizer.save_pretrained(str(out_dir))

    for p in out_dir.iterdir():
        if p.is_dir():
            shutil.rmtree(p)

    _report_size(final_onnx)

def _try_gliner_builtin_export(model: Any, out_dir: Path, fp32_onnx: Path) -> bool:
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

        candidates = list(tmp.rglob("*.onnx"))

        if not candidates:
            shutil.rmtree(tmp, ignore_errors=True)
            return False

        shutil.move(str(candidates[0]), str(fp32_onnx))
        shutil.rmtree(tmp, ignore_errors=True)
        return True

    except Exception as exc:
        print(f"  Built-in export raised {exc!r}, falling back...")
        shutil.rmtree(tmp, ignore_errors=True)
        return False

def _export_gliner_via_torch(model: Any, out_path: Path) -> None:
    import torch

    inner = model.model
    inner.eval()

    device = next(inner.parameters()).device
    max_len = int(getattr(model, "max_len", getattr(model, "max_length", 384)))

    dummy_ids        = torch.zeros(1, max_len, dtype=torch.long, device=device)
    dummy_attn       = torch.ones(1,  max_len, dtype=torch.long, device=device)
    dummy_words_mask = torch.zeros(1, max_len, dtype=torch.long, device=device)
    dummy_words_mask[0, 1] = 1
    dummy_text_len   = torch.tensor([1], dtype=torch.long, device=device)

    torch.onnx.export(
        inner,
        (dummy_ids, dummy_attn, dummy_words_mask, dummy_text_len),
        str(out_path),
        input_names=["input_ids", "attention_mask", "words_mask", "text_lengths"],
        output_names=["logits"],
        dynamic_axes={
            "input_ids": {0: "batch", 1: "seq_len"},
            "attention_mask": {0: "batch", 1: "seq_len"},
            "words_mask": {0: "batch", 1: "seq_len"},
            "text_lengths": {0: "batch"},
            "logits": {0: "batch", 1: "num_words", 2: "num_words"},
        },
        opset_version=14,
        do_constant_folding=True,
    )

def make_archive(model_dir: Path, dest_dir: Path, dir_name: str, archive_name: str) -> Path:
    dest_dir.mkdir(parents=True, exist_ok=True)

    archive_path = dest_dir / archive_name

    print(f"  Packing {archive_name}...")

    with tarfile.open(archive_path, "w:gz", compresslevel=6) as tf:
        for fp in sorted(model_dir.iterdir()):
            if fp.is_file():
                tf.add(str(fp), arcname=f"{dir_name}/{fp.name}")

    print(f"  Archive: {archive_path}")
    return archive_path

def _report_size(p: Path) -> None:
    print(f"  model.onnx: {p.stat().st_size // 1_000_000} MB")

def _die(msg: str) -> None:
    print(f"\nERROR: {msg}", file=sys.stderr)
    sys.exit(1)

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
