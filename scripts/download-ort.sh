#!/usr/bin/env bash
# Downloads the ONNX Runtime shared library for the current platform into ./ort-lib/.
# The file is renamed to the canonical name expected by fquery at runtime.
set -euo pipefail

ORT_VERSION="${ORT_VERSION:-1.20.0}"
BASE="https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}"

OS="$(uname -s)"
ARCH="$(uname -m)"

mkdir -p ort-lib

case "${OS}" in
  Linux)
    case "${ARCH}" in
      x86_64)  PLATFORM="linux-x64" ;;
      aarch64) PLATFORM="linux-aarch64" ;;
      *) echo "Unsupported Linux arch: ${ARCH}" >&2; exit 1 ;;
    esac
    FILE="onnxruntime-${PLATFORM}-${ORT_VERSION}.tgz"
    echo "Downloading ${FILE}..."
    curl -fsSL "${BASE}/${FILE}" \
      | tar -xz -C /tmp "onnxruntime-${PLATFORM}-${ORT_VERSION}/lib/libonnxruntime.so.${ORT_VERSION}"
    cp "/tmp/onnxruntime-${PLATFORM}-${ORT_VERSION}/lib/libonnxruntime.so.${ORT_VERSION}" \
       "ort-lib/libonnxruntime.so"
    rm -rf "/tmp/onnxruntime-${PLATFORM}-${ORT_VERSION}"
    ;;

  Darwin)
    FILE="onnxruntime-osx-universal2-${ORT_VERSION}.tgz"
    echo "Downloading ${FILE}..."
    curl -fsSL "${BASE}/${FILE}" \
      | tar -xz -C /tmp "onnxruntime-osx-universal2-${ORT_VERSION}/lib/libonnxruntime.${ORT_VERSION}.dylib"
    cp "/tmp/onnxruntime-osx-universal2-${ORT_VERSION}/lib/libonnxruntime.${ORT_VERSION}.dylib" \
       "ort-lib/libonnxruntime.dylib"
    rm -rf "/tmp/onnxruntime-osx-universal2-${ORT_VERSION}"
    ;;

  *)
    echo "Unsupported OS: ${OS}" >&2
    exit 1
    ;;
esac

echo "ORT ${ORT_VERSION} ready in ort-lib/"
