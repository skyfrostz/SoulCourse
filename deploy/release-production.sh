#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY_PATH="${SOULCOURSE_BINARY:-$ROOT_DIR/soulcourse-linux-amd64}"
RUNTIME_DATABASE_URL="${DATABASE_URL:-}"
MIGRATION_DATABASE_URL_VALUE="${MIGRATION_DATABASE_URL:-}"

[[ -n "$RUNTIME_DATABASE_URL" ]] || { printf 'DATABASE_URL is required for the runtime role\n' >&2; exit 1; }
[[ -n "$MIGRATION_DATABASE_URL_VALUE" ]] || { printf 'MIGRATION_DATABASE_URL is required for the migration role\n' >&2; exit 1; }

export DATABASE_URL="$RUNTIME_DATABASE_URL"
export SOULCOURSE_BINARY="$BINARY_PATH"

# Validate the exact artifact and runtime configuration before changing the database.
"$ROOT_DIR/deploy/preflight.sh"

DATABASE_URL="$MIGRATION_DATABASE_URL_VALUE" "$ROOT_DIR/deploy/migrate-production.sh"
unset MIGRATION_DATABASE_URL MIGRATION_DATABASE_URL_VALUE SOULCOURSE_BINARY

exec "$BINARY_PATH"
