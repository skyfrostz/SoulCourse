#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

if [ ! -f .env ]; then
  cp .env.example .env
fi

DOCKER_BUILDKIT=1 docker compose up --build
