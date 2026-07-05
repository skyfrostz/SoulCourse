SHELL := /bin/sh
COMPOSE := docker compose

.PHONY: help setup dev dev-detached infra backend frontend frontend-deps test lint prod-build prod-up prod-logs health down logs clean

help:
	@printf "%s\n" "Available commands:"
	@printf "%s\n" "  make setup     Create .env and install frontend dependencies"
	@printf "%s\n" "  make dev       Start PostgreSQL, Redis, Go API, and Vue app"
	@printf "%s\n" "  make dev-detached Start local stack in the background"
	@printf "%s\n" "  make infra     Start PostgreSQL and Redis only"
	@printf "%s\n" "  make backend   Start backend service through Docker Compose"
	@printf "%s\n" "  make frontend  Start frontend service through Docker Compose"
	@printf "%s\n" "  make frontend-deps Refresh frontend node_modules volume"
	@printf "%s\n" "  make test      Run backend and frontend checks"
	@printf "%s\n" "  make prod-build Sequentially build production images"
	@printf "%s\n" "  make prod-up   Start production compose stack"
	@printf "%s\n" "  make health    Check local service health endpoints"
	@printf "%s\n" "  make down      Stop local containers"
	@printf "%s\n" "  make logs      Follow container logs"

setup:
	@test -f .env || cp .env.example .env
	cd frontend && pnpm install

dev:
	@test -f .env || cp .env.example .env
	DOCKER_BUILDKIT=1 $(COMPOSE) up --build

dev-detached:
	@test -f .env || cp .env.example .env
	DOCKER_BUILDKIT=1 $(COMPOSE) up -d --build

infra:
	@test -f .env || cp .env.example .env
	$(COMPOSE) up -d postgres redis

backend:
	@test -f .env || cp .env.example .env
	DOCKER_BUILDKIT=1 $(COMPOSE) up --build backend

frontend:
	DOCKER_BUILDKIT=1 $(COMPOSE) up --build frontend

frontend-deps:
	DOCKER_BUILDKIT=1 $(COMPOSE) run --rm --no-deps frontend pnpm install

test:
	$(COMPOSE) run --rm backend go test ./...
	cd frontend && pnpm build

lint:
	cd frontend && pnpm build
	$(COMPOSE) run --rm backend go test ./...

prod-build:
	DOCKER_BUILDKIT=1 COMPOSE_PARALLEL_LIMIT=1 $(COMPOSE) -f docker-compose.prod.yml build backend
	DOCKER_BUILDKIT=1 COMPOSE_PARALLEL_LIMIT=1 $(COMPOSE) -f docker-compose.prod.yml build frontend

prod-up:
	@test -f .env || cp .env.production.example .env
	$(COMPOSE) -f docker-compose.prod.yml up -d

prod-logs:
	$(COMPOSE) -f docker-compose.prod.yml logs -f

health:
	@curl -fsS http://localhost:$${HTTP_HOST_PORT:-8081}/healthz && printf "\n"
	@curl -fsS http://localhost:$${HTTP_HOST_PORT:-8081}/readyz && printf "\n"
	@curl -fsS http://localhost:$${FRONTEND_HOST_PORT:-$${FRONTEND_PORT:-5173}}/ && printf "\nfrontend ok\n"

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

clean:
	$(COMPOSE) down -v --remove-orphans
