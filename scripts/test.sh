#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

docker compose run --rm backend go test ./...
cd frontend
pnpm build
