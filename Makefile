SHELL := /bin/sh
COMPOSE := docker compose

.PHONY: help setup dev infra backend frontend test lint down logs clean

help:
	@printf "%s\n" "Available commands:"
	@printf "%s\n" "  make setup     Create .env and install frontend dependencies"
	@printf "%s\n" "  make dev       Start PostgreSQL, Redis, Go API, and Vue app"
	@printf "%s\n" "  make infra     Start PostgreSQL and Redis only"
	@printf "%s\n" "  make backend   Start backend service through Docker Compose"
	@printf "%s\n" "  make frontend  Start frontend service through Docker Compose"
	@printf "%s\n" "  make test      Run backend and frontend checks"
	@printf "%s\n" "  make down      Stop local containers"
	@printf "%s\n" "  make logs      Follow container logs"

setup:
	@test -f .env || cp .env.example .env
	cd frontend && pnpm install

dev:
	@test -f .env || cp .env.example .env
	$(COMPOSE) up --build

infra:
	@test -f .env || cp .env.example .env
	$(COMPOSE) up -d postgres redis

backend:
	@test -f .env || cp .env.example .env
	$(COMPOSE) up --build backend

frontend:
	$(COMPOSE) up frontend

test:
	$(COMPOSE) run --rm backend go test ./...
	cd frontend && pnpm build

lint:
	cd frontend && pnpm build
	$(COMPOSE) run --rm backend go test ./...

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

clean:
	$(COMPOSE) down -v --remove-orphans
