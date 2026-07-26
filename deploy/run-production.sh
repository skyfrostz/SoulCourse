#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

export ADMIN_TOKEN="$(<data/admin-token)"
export JWT_SECRET="$(<data/jwt-secret)"

while IFS=: read -r key value; do
  value="${value#"${value%%[![:space:]]*}"}"
  case "$key" in
    "Admin email") export ADMIN_EMAIL="$value" ;;
    "Admin password") export ADMIN_PASSWORD="$value" ;;
  esac
done < /root/soulcourse-admin-credentials.txt

: "${ADMIN_EMAIL:?missing admin email}"
: "${ADMIN_PASSWORD:?missing admin password}"

exec ./soulcourse-linux-amd64
