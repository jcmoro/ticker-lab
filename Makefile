.PHONY: help setup dev down clean build lint format typecheck test test-unit test-functional ci db-migrate db-seed openapi-generate job-ingest docker-build deploy fly-setup fly-logs fly-status fly-console fly-db fly-ingest fly-rollback

.DEFAULT_GOAL := help

help: ## Show available targets
	@echo ""
	@echo "  \033[1mDevelopment\033[0m"
	@echo "  \033[36msetup\033[0m              Build containers and install dependencies"
	@echo "  \033[36mdev\033[0m                Start development environment"
	@echo "  \033[36mdown\033[0m               Stop all containers"
	@echo "  \033[36mclean\033[0m              Remove containers, volumes, and node_modules"
	@echo ""
	@echo "  \033[1mQuality\033[0m"
	@echo "  \033[36mlint\033[0m               Run Biome linter"
	@echo "  \033[36mformat\033[0m             Run Biome formatter"
	@echo "  \033[36mtypecheck\033[0m          TypeScript type checking"
	@echo "  \033[36mci\033[0m                 Run full CI pipeline (lint + typecheck + test)"
	@echo ""
	@echo "  \033[1mTesting\033[0m"
	@echo "  \033[36mtest\033[0m               Run all tests"
	@echo "  \033[36mtest-unit\033[0m          Run unit tests only"
	@echo "  \033[36mtest-functional\033[0m    Run functional tests only"
	@echo ""
	@echo "  \033[1mDatabase\033[0m"
	@echo "  \033[36mdb-migrate\033[0m         Run database migrations"
	@echo "  \033[36mdb-seed\033[0m            Seed development data"
	@echo ""
	@echo "  \033[1mOpenAPI & Jobs\033[0m"
	@echo "  \033[36mopenapi-generate\033[0m   Generate TypeScript types from OpenAPI spec"
	@echo "  \033[36mjob-ingest\033[0m         Trigger daily data ingestion (local)"
	@echo ""
	@echo "  \033[1mBuild & Deploy\033[0m"
	@echo "  \033[36mbuild\033[0m              Build for production"
	@echo "  \033[36mdocker-build\033[0m       Build production Docker image"
	@echo "  \033[36mdeploy\033[0m             Trigger Render deploy"
	@echo ""
	@echo "  \033[1mProduction (Neon + Render)\033[0m"
	@echo "  \033[36mprod-db\033[0m            Connect to Neon Postgres"
	@echo "  \033[36mprod-ingest\033[0m        Run daily ingestion against production DB"
	@echo "  \033[36mprod-backfill\033[0m      Backfill historical rates against production DB"
	@echo ""

# ─── Development ─────────────────────────────────────────────

setup:
	cp -n .env.example .env 2>/dev/null || true
	docker compose build
	docker compose run --rm api pnpm install

dev:
	docker compose up

down:
	docker compose down

clean:
	docker compose down -v
	rm -rf node_modules apps/*/node_modules packages/*/node_modules

# ─── Quality ─────────────────────────────────────────────────

lint:
	docker compose run --rm api pnpm lint

format:
	docker compose run --rm api pnpm format

typecheck:
	docker compose run --rm api pnpm typecheck

ci: lint typecheck test

# ─── Testing ─────────────────────────────────────────────────

test:
	docker compose run --rm api pnpm test

test-unit:
	docker compose run --rm api pnpm test:unit

test-functional:
	docker compose run --rm api pnpm test:functional

# ─── Database ────────────────────────────────────────────────

db-migrate:
	docker compose run --rm api pnpm --filter @ticker-lab/api db:migrate

db-seed:
	docker compose run --rm api pnpm --filter @ticker-lab/api db:seed

# ─── OpenAPI & Jobs ──────────────────────────────────────────

openapi-generate:
	docker compose run --rm api pnpm --filter @ticker-lab/api openapi:generate

job-ingest:
	docker compose run --rm api pnpm --filter @ticker-lab/api job:ingest

job-backfill: ## Backfill historical rates (default: 2024-01-01 to today)
	docker compose run --rm api pnpm --filter @ticker-lab/api job:backfill

# ─── Build & Deploy ──────────────────────────────────────────

build:
	docker compose run --rm api pnpm --filter @ticker-lab/api build

docker-build:
	docker build -f docker/api/Dockerfile --target prod -t ticker-lab-api .

deploy: ## Trigger Render deploy (requires RENDER_DEPLOY_HOOK env var)
	@curl -s -X POST "$$RENDER_DEPLOY_HOOK" && echo "Deploy triggered"

# ─── Production (Neon + Render) ──────────────────────────────

prod-db: ## Connect to Neon Postgres
	@psql "$$DATABASE_URL"

prod-ingest: ## Run daily ingestion against production DB
	DATABASE_URL="$$DATABASE_URL" pnpm --filter @ticker-lab/api job:ingest

prod-backfill: ## Backfill historical rates against production DB
	DATABASE_URL="$$DATABASE_URL" pnpm --filter @ticker-lab/api job:backfill
