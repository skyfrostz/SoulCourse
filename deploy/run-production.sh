#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

"$ROOT_DIR/deploy/preflight.sh"
exec "$ROOT_DIR/soulcourse-linux-amd64"
