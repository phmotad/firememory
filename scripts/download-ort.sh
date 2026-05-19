#!/usr/bin/env bash
# Downloads the ONNX Runtime shared library for the current platform.
# Output goes into ort-lib/<os>/ so goreleaser can template ort-lib/{{ .Os }}/*.
set -euo pipefail

ORT_VERSION="${ORT_VERSION:-1.25.0}"
BASE="https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
  Linux)
    case "${ARCH}" in
      x86_64)  PLATFORM="linux-x64" ;;
      aarch64) PLATFORM="linux-aarch64" ;;
      *) echo "Unsupported Linux arch: ${ARCH}" >&2; exit 1 ;;
    esac
    DEST="ort-lib/linux"
    mkdir -p "${DEST}"
    FILE="onnxruntime-${PLATFORM}-${ORT_VERSION}.tgz"
    echo "Downloading ${FILE}..."
    curl -fsSL "${BASE}/${FILE}" \
      | tar -xz -C /tmp "onnxruntime-${PLATFORM}-${ORT_VERSION}/lib/libonnxruntime.so.${ORT_VERSION}"
    cp "/tmp/onnxruntime-${PLATFORM}-${ORT_VERSION}/lib/libonnxruntime.so.${ORT_VERSION}" \
       "${DEST}/libonnxruntime.so"
    rm -rf "/tmp/onnxruntime-${PLATFORM}-${ORT_VERSION}"
    ;;

  Darwin)
    DEST="ort-lib/darwin"
    mkdir -p "${DEST}"
    FILE="onnxruntime-osx-arm64-${ORT_VERSION}.tgz"
    echo "Downloading ${FILE}..."
    curl -fsSL "${BASE}/${FILE}" \
      | tar -xz -C /tmp "onnxruntime-osx-arm64-${ORT_VERSION}/lib/libonnxruntime.${ORT_VERSION}.dylib"
    cp "/tmp/onnxruntime-osx-arm64-${ORT_VERSION}/lib/libonnxruntime.${ORT_VERSION}.dylib" \
       "${DEST}/libonnxruntime.dylib"
    rm -rf "/tmp/onnxruntime-osx-arm64-${ORT_VERSION}"
    ;;

  *)
    echo "Unsupported OS: ${OS}" >&2
    exit 1
    ;;
esac

echo "ORT ${ORT_VERSION} ready in ${DEST}/"
