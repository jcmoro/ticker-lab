.PHONY: help setup dev down build test test-unit test-functional lint format typecheck ci db-migrate db-seed openapi-generate job-ingest docker-build deploy clean

.DEFAULT_GOAL := help

# ─── Environment ──────────────────────────────────────────────

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

setup: ## Build containers and install dependencies
	cp -n .env.example .env 2>/dev/null || true
	docker compose build
	docker compose run --rm api pnpm install

dev: ## Start development environment
	docker compose up

down: ## Stop all containers
	docker compose down

clean: ## Remove containers, volumes, and node_modules
	docker compose down -v
	rm -rf node_modules apps/*/node_modules packages/*/node_modules

# ─── Build ────────────────────────────────────────────────────

build: ## Build for production
	docker compose run --rm api pnpm --filter @ticker-lab/api build

# ─── Quality ──────────────────────────────────────────────────

lint: ## Run Biome linter
	docker compose run --rm api pnpm lint

format: ## Run Biome formatter
	docker compose run --rm api pnpm format

typecheck: ## TypeScript type checking
	docker compose run --rm api pnpm typecheck

# ─── Testing ──────────────────────────────────────────────────

test: ## Run all tests
	docker compose run --rm api pnpm test

test-unit: ## Run unit tests only
	docker compose run --rm api pnpm test:unit

test-functional: ## Run functional tests only
	docker compose run --rm api pnpm test:functional

# ─── Database ─────────────────────────────────────────────────

db-migrate: ## Run database migrations
	docker compose run --rm api pnpm --filter @ticker-lab/api db:migrate

db-seed: ## Seed development data
	docker compose run --rm api pnpm --filter @ticker-lab/api db:seed

# ─── OpenAPI ──────────────────────────────────────────────────

openapi-generate: ## Generate TypeScript types from OpenAPI spec
	docker compose run --rm api pnpm --filter @ticker-lab/api openapi:generate

# ─── Jobs ─────────────────────────────────────────────────────

job-ingest: ## Manually trigger daily data ingestion
	docker compose run --rm api pnpm --filter @ticker-lab/api job:ingest

# ─── CI ───────────────────────────────────────────────────────

ci: lint typecheck test ## Run full CI pipeline locally

# ─── Docker ───────────────────────────────────────────────────

docker-build: ## Build production Docker image
	docker build -f docker/api/Dockerfile --target prod -t ticker-lab-api .

# ─── Deploy ───────────────────────────────────────────────────

deploy: ## Deploy to Fly.io
	fly deploy
