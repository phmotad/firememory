#!/usr/bin/env bash
# Installs fmem and fquery from the latest FireMemory GitHub Release.
#
#   curl -fsSL https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.sh | bash
#   INSTALL_DIR=/usr/local/bin bash install.sh   # custom directory
set -euo pipefail

REPO="phmotad/firememory"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# ── Detect platform ─────────────────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)
    echo "error: unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

case "${OS}" in
  linux | darwin) ;;
  *)
    echo "error: unsupported OS: ${OS} — use install.ps1 on Windows" >&2
    exit 1
    ;;
esac

# ── Resolve latest version ───────────────────────────────────────────────────

VERSION="${VERSION:-}"
if [ -z "${VERSION}" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')"
fi

if [ -z "${VERSION}" ]; then
  echo "error: could not determine latest release version" >&2
  exit 1
fi

echo "Installing FireMemory v${VERSION} (${OS}/${ARCH})..."

# ── Download and verify ──────────────────────────────────────────────────────

ARCHIVE="firememory_${VERSION}_${OS}_${ARCH}.tar.gz"
SUMS="SHA256SUMS"
BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

curl -fsSL --progress-bar "${BASE_URL}/${ARCHIVE}" -o "${TMPDIR}/${ARCHIVE}"
curl -fsSL "${BASE_URL}/${SUMS}" -o "${TMPDIR}/${SUMS}"

# Verify checksum.
( cd "${TMPDIR}" && grep "${ARCHIVE}" "${SUMS}" | sha256sum --check --quiet )
echo "checksum OK"

tar -xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"

# ── Install binaries ─────────────────────────────────────────────────────────

mkdir -p "${INSTALL_DIR}"

install -m 0755 "${TMPDIR}/fmem"   "${INSTALL_DIR}/fmem"
install -m 0755 "${TMPDIR}/fquery" "${INSTALL_DIR}/fquery"

# Install the ONNX Runtime shared library next to fquery.
case "${OS}" in
  linux)  ORT_LIB="libonnxruntime.so" ;;
  darwin) ORT_LIB="libonnxruntime.dylib" ;;
esac
if [ -f "${TMPDIR}/${ORT_LIB}" ]; then
  install -m 0644 "${TMPDIR}/${ORT_LIB}" "${INSTALL_DIR}/${ORT_LIB}"
fi

# ── Done ─────────────────────────────────────────────────────────────────────

echo ""
echo "Installed:"
echo "  ${INSTALL_DIR}/fmem"
echo "  ${INSTALL_DIR}/fquery"
echo ""

# Warn if INSTALL_DIR is not in PATH.
if ! echo ":${PATH}:" | grep -q ":${INSTALL_DIR}:"; then
  echo "  Add to PATH: export PATH=\"${INSTALL_DIR}:\$PATH\""
  echo ""
fi

echo "Next steps:"
echo "  fmem init ~/my.fbrain         # create a brainfile"
echo "  fquery mcp                    # start MCP server (downloads models once)"
