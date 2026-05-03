#!/usr/bin/env bash
# Downloads prebuilt libtokenizers.a for linux amd64 and aarch64.
# Output: tokenizers-lib/linux-amd64/libtokenizers.a
#          tokenizers-lib/linux-aarch64/libtokenizers.a
set -euo pipefail

BASE="https://github.com/daulet/tokenizers/releases/latest/download"

download_for() {
  local ARCH="$1"     # e.g. amd64, aarch64
  local LABEL="$2"    # e.g. linux-amd64, linux-aarch64
  local DEST="tokenizers-lib/${LABEL}"

  mkdir -p "${DEST}"
  echo "Downloading libtokenizers for ${LABEL}..."
  curl -fsSL "${BASE}/libtokenizers.linux-${ARCH}.tar.gz" \
    | tar -xz -C "${DEST}"
  echo "  → ${DEST}/libtokenizers.a"
}

download_for "x86_64"  "linux-amd64"
download_for "aarch64" "linux-aarch64"

echo "libtokenizers ready in tokenizers-lib/"
