#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
BINARY_PATH="${SOULCOURSE_BINARY:-$ROOT_DIR/soulcourse-linux-amd64}"

fail() {
  printf 'production preflight failed: %s\n' "$1" >&2
  exit 1
}

[[ "${APP_ENV:-}" == "production" || "${APP_ENV:-}" == "prod" ]] || fail "APP_ENV must be production"
[[ "$(id -u)" -ne 0 ]] || fail "service must not run as root"
[[ -x "$BINARY_PATH" ]] || fail "production binary is missing or not executable"
if [[ -n "${SOULCOURSE_CLEANUP_BINARY:-}" ]]; then
  [[ -x "$SOULCOURSE_CLEANUP_BINARY" ]] || fail "upload cleanup binary is missing or not executable"
fi

JWT_SECRET_VALUE="${JWT_SECRET:-}"
METRICS_TOKEN_VALUE="${METRICS_TOKEN:-}"
[[ ${#JWT_SECRET_VALUE} -ge 32 ]] || fail "JWT_SECRET must contain at least 32 characters"
[[ "$JWT_SECRET_VALUE" != "replace-me-before-production" ]] || fail "JWT_SECRET still uses the default value"
[[ ${#METRICS_TOKEN_VALUE} -ge 32 ]] || fail "METRICS_TOKEN must contain at least 32 characters"
[[ -n "${TRUSTED_PROXIES:-}" ]] || fail "TRUSTED_PROXIES is required"
[[ -n "${CORS_ALLOWED_ORIGINS:-}" ]] || fail "CORS_ALLOWED_ORIGINS is required"
[[ "$CORS_ALLOWED_ORIGINS" != *"*"* ]] || fail "wildcard CORS is forbidden"
[[ -z "${ADMIN_PASSWORD:-}" ]] || fail "ADMIN_PASSWORD plaintext is forbidden; use ADMIN_PASSWORD_HASH"
[[ -n "${ADMIN_EMAIL:-}" ]] || fail "ADMIN_EMAIL is required"
[[ -n "${ADMIN_PASSWORD_HASH:-}" ]] || fail "ADMIN_PASSWORD_HASH is required"
[[ "${ADMIN_ROLE:-}" == "super_admin" || "${ADMIN_ROLE:-}" == "content_editor" || "${ADMIN_ROLE:-}" == "moderator" ]] || fail "ADMIN_ROLE is invalid"
[[ -n "${SMTP_HOST:-}" ]] || fail "SMTP_HOST is required for registration"
[[ -n "${SMTP_USERNAME:-}" ]] || fail "SMTP_USERNAME is required for registration"
[[ -n "${SMTP_PASSWORD:-}" ]] || fail "SMTP_PASSWORD is required for registration"
[[ -n "${SMTP_FROM_EMAIL:-}" ]] || fail "SMTP_FROM_EMAIL is required for registration"
[[ "${SMTP_USE_TLS:-false}" == "true" || "${SMTP_STARTTLS:-false}" == "true" ]] || fail "SMTP must use TLS or STARTTLS"

[[ "${DATABASE_DRIVER:-}" == "postgres" ]] || fail "DATABASE_DRIVER must be postgres"
[[ "${DATABASE_URL:-}" == postgres://* || "${DATABASE_URL:-}" == postgresql://* ]] || fail "DATABASE_URL must be a PostgreSQL URL"
[[ "${DATABASE_URL:-}" == *"sslmode="* ]] || fail "DATABASE_URL must declare sslmode explicitly"
[[ "${DATABASE_MAX_OPEN_CONNS:-20}" =~ ^[0-9]+$ ]] || fail "DATABASE_MAX_OPEN_CONNS must be an integer"
(( DATABASE_MAX_OPEN_CONNS > 0 && DATABASE_MAX_OPEN_CONNS <= 20 )) || fail "DATABASE_MAX_OPEN_CONNS must be between 1 and 20"

[[ "${STORAGE_DRIVER:-}" == "s3" ]] || fail "STORAGE_DRIVER must be s3"
[[ "${S3_ENDPOINT:-}" == https://* ]] || fail "S3_ENDPOINT must use https"
[[ -n "${S3_BUCKET:-}" ]] || fail "S3_BUCKET is required"
[[ -n "${S3_REGION:-}" ]] || fail "S3_REGION is required"
[[ "${S3_CDN_BASE_URL:-}" == https://* ]] || fail "S3_CDN_BASE_URL must use https"

if [[ -n "${FRONTEND_DIST_DIR:-}" ]]; then
  [[ -f "$FRONTEND_DIST_DIR/index.html" ]] || fail "FRONTEND_DIST_DIR does not contain index.html"
fi

printf 'production preflight passed\n'
