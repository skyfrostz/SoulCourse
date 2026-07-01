#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

if [ ! -f .env ]; then
  cp .env.example .env
fi

docker compose up -d postgres redis
