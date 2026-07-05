#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

# Keep production builds stable on small VPS instances by avoiding
# concurrent Go and Node builds.
export DOCKER_BUILDKIT=1
export COMPOSE_PARALLEL_LIMIT=1

docker compose -f docker-compose.prod.yml build backend
docker compose -f docker-compose.prod.yml build frontend
