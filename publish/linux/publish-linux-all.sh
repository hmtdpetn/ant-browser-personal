#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for arch in amd64 arm64; do
  echo "===== Publishing linux/$arch ====="
  bash "$SCRIPT_DIR/publish-linux.sh" --arch "$arch" "$@"
done
