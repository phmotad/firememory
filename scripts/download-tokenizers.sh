#!/usr/bin/env bash
# Downloads prebuilt libtokenizers.a for the current platform.
# Output: tokenizers-lib/<os>-<arch>/libtokenizers.a
set -euo pipefail

BASE="https://github.com/daulet/tokenizers/releases/latest/download"
OS="$(uname -s)"

mkdir -p tokenizers-lib

case "${OS}" in
  Linux)
    for ARCH in x86_64 aarch64; do
      LABEL="linux-$([ "${ARCH}" = "x86_64" ] && echo "amd64" || echo "aarch64")"
      DEST="tokenizers-lib/${LABEL}"
      mkdir -p "${DEST}"
      echo "Downloading libtokenizers for linux-${ARCH}..."
      curl -fsSL "${BASE}/libtokenizers.linux-${ARCH}.tar.gz" | tar -xz -C "${DEST}"
      echo "  → ${DEST}/libtokenizers.a"
    done
    ;;

  Darwin)
    for ARCH in arm64 x86_64; do
      LABEL="darwin-$([ "${ARCH}" = "x86_64" ] && echo "amd64" || echo "arm64")"
      DEST="tokenizers-lib/${LABEL}"
      mkdir -p "${DEST}"
      echo "Downloading libtokenizers for darwin-${ARCH}..."
      curl -fsSL "${BASE}/libtokenizers.darwin-${ARCH}.tar.gz" | tar -xz -C "${DEST}"
      echo "  → ${DEST}/libtokenizers.a"
    done
    ;;

  *)
    echo "No tokenizers download needed for ${OS}"
    ;;
esac

echo "libtokenizers ready in tokenizers-lib/"
