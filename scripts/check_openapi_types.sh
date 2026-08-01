#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GENERATED_FILE="frontend/src/lib/generated/openapi-types.ts"

cd "$ROOT_DIR"

GOCACHE="${GOCACHE:-$ROOT_DIR/.cache/go-build}" go -C backend run ./cmd/openapi-types

if ! git diff --exit-code -- "$GENERATED_FILE" >/dev/null; then
  echo "OpenAPI generated types are out of date. Run: go -C backend run ./cmd/openapi-types" >&2
  git diff -- "$GENERATED_FILE" >&2
  exit 1
fi
