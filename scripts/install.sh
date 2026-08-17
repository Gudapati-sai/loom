#!/usr/bin/env bash
# install.sh — one-command installer for loom (bash / macOS / Linux).
#
# Builds loom from the GitHub source using the committed go.sum, so it works
# even where `go install @latest` fails on checksum-DB/proxy issues for a
# brand-new module. Installs to $(go env GOPATH)/bin/loom and prints the
# PATH line if the Go bin dir isn't already on PATH.
#
# Usage:
#   git clone https://github.com/Gudapati-sai/loom.git && cd loom && bash scripts/install.sh
#   # optional version: bash scripts/install.sh v1.0.0

set -euo pipefail
VERSION="${1:-v1.0.0}"

command -v git >/dev/null 2>&1 || { echo "git is required — install it from https://git-scm.com" >&2; exit 1; }
command -v go  >/dev/null 2>&1 || { echo "Go is required — install it from https://go.dev/dl" >&2; exit 1; }

BIN_DIR="$(go env GOPATH)/bin"
mkdir -p "$BIN_DIR"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

git clone -q --depth 1 --branch "$VERSION" https://github.com/Gudapati-sai/loom.git "$TMP"
(cd "$TMP" && go build -o "$BIN_DIR/loom" .)

echo ""
echo "  installed loom to $BIN_DIR/loom"
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  echo "  The Go bin directory is not on your PATH yet:"
  echo "    export PATH=\"\$PATH:$BIN_DIR\""
fi
echo ""
echo "  Run:  loom help"
