#!/usr/bin/env bash
# Build wsl_proxy.exe (Windows amd64) using Docker.
# Output: wails-app/output/wsl_proxy.exe
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE="wsl-proxy-wails-builder"
OUTPUT_DIR="$SCRIPT_DIR/output"

echo "==> Building Docker image (builder stage)..."
# Target the 'builder' stage — the final 'scratch' stage has no shell/entrypoint
# so docker create/cp would fail with "no command specified".
# --no-cache ensures all layers are rebuilt (avoids stale embedded assets).
docker build --no-cache --target builder -t "$IMAGE" "$SCRIPT_DIR"

echo "==> Extracting wsl_proxy.exe..."
mkdir -p "$OUTPUT_DIR"

CONTAINER=$(docker create "$IMAGE" sh)
docker cp "$CONTAINER:/output/wsl_proxy.exe" "$OUTPUT_DIR/wsl_proxy.exe"
docker rm "$CONTAINER" > /dev/null

echo "==> Done: $OUTPUT_DIR/wsl_proxy.exe"
