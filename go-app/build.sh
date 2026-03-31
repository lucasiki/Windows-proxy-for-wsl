#!/usr/bin/env bash
# Cross-compile wsl_proxy.exe from Linux using Docker.
# Usage: ./build.sh [output-dir]
#   output-dir defaults to ./output

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT_DIR="${1:-$SCRIPT_DIR/output}"
mkdir -p "$OUTPUT_DIR"

echo "==> Building and extracting wsl_proxy.exe ..."
docker build \
  --target artifact \
  --output "type=local,dest=$OUTPUT_DIR" \
  "$SCRIPT_DIR"

echo "==> Done: $OUTPUT_DIR/wsl_proxy.exe"
