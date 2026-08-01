#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MIGRATION_DIR="$ROOT_DIR/backend/migrations/postgres"
GOOSE_BIN="${GOOSE_BIN:-goose}"
EXPECTED_VERSION="$(find "$MIGRATION_DIR" -maxdepth 1 -type f -name '*.sql' -exec basename {} \; | sed -E 's/^0*([0-9]+)_.*/\1/' | sort -n | tail -1)"

[[ "${APP_ENV:-}" == "production" || "${APP_ENV:-}" == "prod" ]] || { printf 'APP_ENV must be production\n' >&2; exit 1; }
[[ -n "${DATABASE_URL:-}" ]] || { printf 'DATABASE_URL is required\n' >&2; exit 1; }
[[ -n "${GUANGDONG_DATA_YEAR:-}" ]] || { printf 'GUANGDONG_DATA_YEAR is required\n' >&2; exit 1; }
[[ -n "$EXPECTED_VERSION" ]] || { printf 'no PostgreSQL migrations found\n' >&2; exit 1; }
command -v "$GOOSE_BIN" >/dev/null 2>&1 || { printf 'goose binary is required: %s\n' "$GOOSE_BIN" >&2; exit 1; }
command -v psql >/dev/null 2>&1 || { printf 'psql is required\n' >&2; exit 1; }

GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DATABASE_URL" GOOSE_MIGRATION_DIR="$MIGRATION_DIR" "$GOOSE_BIN" up

actual="$({ psql "$DATABASE_URL" -XAtv ON_ERROR_STOP=1 -c \
  "SELECT version_id::text || ':' || is_applied::text FROM goose_db_version ORDER BY id DESC LIMIT 1"; } 2>/dev/null)"
[[ "$actual" == "$EXPECTED_VERSION:true" ]] || {
  printf 'schema version check failed: got %s, expected %s:true\n' "$actual" "$EXPECTED_VERSION" >&2
  exit 1
}

psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -v target_year="$GUANGDONG_DATA_YEAR" \
  -f "$ROOT_DIR/scripts/postgres/validate-guangdong-production.sql"

printf 'production migration and Guangdong data gates passed: version=%s year=%s\n' "$EXPECTED_VERSION" "$GUANGDONG_DATA_YEAR"
