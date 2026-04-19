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
	@echo "  \033[36mdeploy\033[0m             Deploy to Fly.io"
	@echo ""
	@echo "  \033[1mFly.io Operations\033[0m"
	@echo "  \033[36mfly-setup\033[0m          First-time Fly.io setup (app + Postgres)"
	@echo "  \033[36mfly-status\033[0m         Show app status and machines"
	@echo "  \033[36mfly-logs\033[0m           Tail production logs"
	@echo "  \033[36mfly-console\033[0m        Open SSH console in production"
	@echo "  \033[36mfly-db\033[0m             Connect to production Postgres"
	@echo "  \033[36mfly-ingest\033[0m         Run ingestion job in production"
	@echo "  \033[36mfly-rollback\033[0m       Show releases (pick one to rollback)"
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

# ─── Build & Deploy ──────────────────────────────────────────

build:
	docker compose run --rm api pnpm --filter @ticker-lab/api build

docker-build:
	docker build -f docker/api/Dockerfile --target prod -t ticker-lab-api .

deploy:
	fly deploy

# ─── Fly.io Operations ──────────────────────────────────────

fly-setup:
	fly launch --no-deploy
	fly postgres create --name tickerlab-db --region mad --vm-size shared-cpu-1x --volume-size 1
	fly postgres attach tickerlab-db
	@echo "Done. Run 'make deploy' to deploy, then 'make fly-ingest' to seed data."

fly-status:
	fly status

fly-logs:
	fly logs

fly-console:
	fly ssh console

fly-db:
	fly postgres connect -a tickerlab-db

fly-ingest:
	fly ssh console -C "node dist/infrastructure/jobs/ingest.js"

fly-rollback:
	fly releases
