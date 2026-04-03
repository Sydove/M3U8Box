#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_PATH="${1:-$ROOT_DIR/M3U8Box}"

cd "$ROOT_DIR"

echo "Building M3U8Box..."
echo "Output: $OUTPUT_PATH"

go build -o "$OUTPUT_PATH" ./cmd/M3U8Box

echo "Build completed."
